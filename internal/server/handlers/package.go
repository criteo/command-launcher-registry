package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/criteo/command-launcher-registry/internal/apierrors"
	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/criteo/command-launcher-registry/internal/roles"
	"github.com/criteo/command-launcher-registry/internal/storage"
)

// PackageHandler handles package CRUD operations. Each write is checked
// against the caller's roles by the authorizer's guard, under the storage
// lock. With requireMaintainers on, a write that leaves a package with no
// maintainers is rejected with 400.
type PackageHandler struct {
	store              storage.Store
	authorizer         *Authorizer
	requireMaintainers bool
	logger             *slog.Logger
}

// NewPackageHandler creates a new package handler. requireMaintainers turns
// on the required-maintainers rule (setting require_maintainers).
func NewPackageHandler(store storage.Store, authorizer *Authorizer, requireMaintainers bool, logger *slog.Logger) *PackageHandler {
	return &PackageHandler{
		store:              store,
		authorizer:         authorizer,
		requireMaintainers: requireMaintainers,
		logger:             logger,
	}
}

// CreatePackage handles POST /api/v1/registry/:name/package
func (h *PackageHandler) CreatePackage(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "name")

	var pkg models.Package

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
		h.logger.Warn("Failed to decode package creation request",
			"registry", registryName,
			"error", err,
			"remote_addr", r.RemoteAddr)
		apierrors.WriteError(w, apierrors.ErrCodeValidationError, "Invalid JSON in request body", http.StatusBadRequest, nil)
		return
	}

	// Validate package, and the required-maintainers rule when it is on
	err := models.ValidatePackage(&pkg)
	if err == nil && h.requireMaintainers {
		err = models.ValidateRequiredMaintainers(pkg.Maintainers)
	}
	if err != nil {
		h.logger.Warn("Package validation failed",
			"registry", registryName,
			"package", pkg.Name,
			"error", err,
			"remote_addr", r.RemoteAddr)
		apierrors.WriteError(w, apierrors.ErrCodeValidationError, err.Error(), http.StatusBadRequest, nil)
		return
	}

	// Initialize versions map if nil
	if pkg.Versions == nil {
		pkg.Versions = make(map[string]*models.Version)
	}

	// Create package: needs registry admin. The caller is not added to
	// maintainers; the audit line logs the maintainers the package starts with.
	guard := h.authorizer.Guard(r, roles.CreatePackage, roles.Target{Registry: registryName, Package: pkg.Name}, &pkg.Maintainers)
	if err := h.store.CreatePackage(r.Context(), registryName, &pkg, guard); err != nil {
		var forbidden *storage.ForbiddenError
		if errors.As(err, &forbidden) {
			apierrors.WriteForbidden(w, forbidden)
			return
		}
		if err == storage.ErrNotFound {
			code, msg, status := apierrors.MapStorageError(err, "registry")
			apierrors.WriteError(w, code, msg, status, nil)
			return
		}
		if err == storage.ErrAlreadyExists {
			code, msg, status := apierrors.MapStorageError(err, "package")
			apierrors.WriteError(w, code, msg, status, nil)
			return
		}

		h.logger.Error("Failed to create package",
			"registry", registryName,
			"package", pkg.Name,
			"error", err)
		apierrors.WriteError(w, apierrors.ErrCodeStorageUnavailable, "Failed to create package", http.StatusInternalServerError, nil)
		return
	}

	// Log successful creation
	h.logger.Info("Package created",
		"registry", registryName,
		"package", pkg.Name,
		"maintainer_count", len(pkg.Maintainers),
		"custom_values", len(pkg.CustomValues),
		"remote_addr", r.RemoteAddr)

	// Return created package
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pkg)
}

// GetPackage handles GET /api/v1/registry/:name/package/:package
func (h *PackageHandler) GetPackage(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "name")
	packageName := chi.URLParam(r, "package")

	// Get package from storage
	pkg, err := h.store.GetPackage(r.Context(), registryName, packageName)
	if err != nil {
		if err == storage.ErrNotFound {
			// Determine if registry or package not found
			if _, regErr := h.store.GetRegistry(r.Context(), registryName); regErr == storage.ErrNotFound {
				code, msg, status := apierrors.MapStorageError(err, "registry")
				apierrors.WriteError(w, code, msg, status, nil)
			} else {
				code, msg, status := apierrors.MapStorageError(err, "package")
				apierrors.WriteError(w, code, msg, status, nil)
			}
			return
		}

		h.logger.Error("Failed to get package",
			"registry", registryName,
			"package", packageName,
			"error", err)
		apierrors.WriteError(w, apierrors.ErrCodeStorageUnavailable, "Failed to retrieve package", http.StatusInternalServerError, nil)
		return
	}

	// Log retrieval
	h.logger.Debug("Package retrieved",
		"registry", registryName,
		"package", packageName,
		"version_count", len(pkg.Versions))

	// Return package
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pkg)
}

// UpdatePackage handles PUT /api/v1/registry/:name/package/:package
func (h *PackageHandler) UpdatePackage(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "name")
	packageName := chi.URLParam(r, "package")

	// Parse request body: a partial update, absent fields keep their value
	var update models.PackageUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		h.logger.Warn("Failed to decode package update request",
			"registry", registryName,
			"package", packageName,
			"error", err,
			"remote_addr", r.RemoteAddr)
		apierrors.WriteError(w, apierrors.ErrCodeValidationError, "Invalid JSON in request body", http.StatusBadRequest, nil)
		return
	}

	// Validate the fields present (and name, if given, against the URL).
	// With the required-maintainers rule on, maintainers may not be set to
	// []; omitting them keeps the stored list and is allowed.
	err := update.Validate(packageName)
	if err == nil && h.requireMaintainers && update.Maintainers != nil {
		err = models.ValidateRequiredMaintainers(*update.Maintainers)
	}
	if err != nil {
		h.logger.Warn("Package validation failed",
			"registry", registryName,
			"package", packageName,
			"error", err,
			"remote_addr", r.RemoteAddr)
		apierrors.WriteError(w, apierrors.ErrCodeValidationError, err.Error(), http.StatusBadRequest, nil)
		return
	}

	// Update package (merged under the storage lock, versions preserved):
	// needs package maintainer.
	guard := h.authorizer.Guard(r, roles.UpdatePackage, roles.Target{Registry: registryName, Package: packageName}, update.Maintainers)
	pkg, err := h.store.UpdatePackage(r.Context(), registryName, packageName, &update, guard)
	if err != nil {
		var forbidden *storage.ForbiddenError
		if errors.As(err, &forbidden) {
			apierrors.WriteForbidden(w, forbidden)
			return
		}
		if err == storage.ErrNotFound {
			// Determine if registry or package not found
			if _, regErr := h.store.GetRegistry(r.Context(), registryName); regErr == storage.ErrNotFound {
				code, msg, status := apierrors.MapStorageError(err, "registry")
				apierrors.WriteError(w, code, msg, status, nil)
			} else {
				code, msg, status := apierrors.MapStorageError(err, "package")
				apierrors.WriteError(w, code, msg, status, nil)
			}
			return
		}

		h.logger.Error("Failed to update package",
			"registry", registryName,
			"package", packageName,
			"error", err)
		apierrors.WriteError(w, apierrors.ErrCodeStorageUnavailable, "Failed to update package", http.StatusInternalServerError, nil)
		return
	}

	// Log successful update
	h.logger.Info("Package updated",
		"registry", registryName,
		"package", pkg.Name,
		"maintainer_count", len(pkg.Maintainers),
		"custom_values", len(pkg.CustomValues),
		"remote_addr", r.RemoteAddr)

	// Return updated package
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pkg)
}

// DeletePackage handles DELETE /api/v1/registry/:name/package/:package
func (h *PackageHandler) DeletePackage(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "name")
	packageName := chi.URLParam(r, "package")

	// Delete package (cascade delete handled by storage layer): needs
	// registry admin.
	guard := h.authorizer.Guard(r, roles.DeletePackage, roles.Target{Registry: registryName, Package: packageName}, nil)
	if err := h.store.DeletePackage(r.Context(), registryName, packageName, guard); err != nil {
		var forbidden *storage.ForbiddenError
		if errors.As(err, &forbidden) {
			apierrors.WriteForbidden(w, forbidden)
			return
		}
		if err == storage.ErrNotFound {
			// Determine if registry or package not found
			if _, regErr := h.store.GetRegistry(r.Context(), registryName); regErr == storage.ErrNotFound {
				code, msg, status := apierrors.MapStorageError(err, "registry")
				apierrors.WriteError(w, code, msg, status, nil)
			} else {
				code, msg, status := apierrors.MapStorageError(err, "package")
				apierrors.WriteError(w, code, msg, status, nil)
			}
			return
		}

		h.logger.Error("Failed to delete package",
			"registry", registryName,
			"package", packageName,
			"error", err)
		apierrors.WriteError(w, apierrors.ErrCodeStorageUnavailable, "Failed to delete package", http.StatusInternalServerError, nil)
		return
	}

	// Log successful deletion
	h.logger.Info("Package deleted",
		"registry", registryName,
		"package", packageName,
		"remote_addr", r.RemoteAddr)

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// ListPackages handles GET /api/v1/registry/:name/package
func (h *PackageHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "name")

	// Get all packages from storage
	packages, err := h.store.ListPackages(r.Context(), registryName)
	if err != nil {
		if err == storage.ErrNotFound {
			code, msg, status := apierrors.MapStorageError(err, "registry")
			apierrors.WriteError(w, code, msg, status, nil)
			return
		}

		h.logger.Error("Failed to list packages",
			"registry", registryName,
			"error", err)
		apierrors.WriteError(w, apierrors.ErrCodeStorageUnavailable, "Failed to list packages", http.StatusInternalServerError, nil)
		return
	}

	// Log retrieval
	h.logger.Debug("Packages listed",
		"registry", registryName,
		"count", len(packages))

	// Return packages
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(packages)
}
