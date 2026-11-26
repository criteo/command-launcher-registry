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

// RegistryHandler handles registry CRUD operations. Writes are checked
// against the caller's roles by the authorizer's guard, under the storage
// lock.
type RegistryHandler struct {
	store      storage.Store
	authorizer *Authorizer
	logger     *slog.Logger
}

// NewRegistryHandler creates a new registry handler
func NewRegistryHandler(store storage.Store, authorizer *Authorizer, logger *slog.Logger) *RegistryHandler {
	return &RegistryHandler{
		store:      store,
		authorizer: authorizer,
		logger:     logger,
	}
}

// CreateRegistry handles POST /api/v1/registry
func (h *RegistryHandler) CreateRegistry(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("CreateRegistry handler called",
		"method", r.Method,
		"content_type", r.Header.Get("Content-Type"),
		"content_length", r.ContentLength,
		"remote_addr", r.RemoteAddr)

	var registry models.Registry

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&registry); err != nil {
		h.logger.Warn("Failed to decode registry creation request",
			"error", err,
			"remote_addr", r.RemoteAddr)
		apierrors.WriteError(w, apierrors.ErrCodeValidationError, "Invalid JSON in request body", http.StatusBadRequest, nil)
		return
	}

	// Validate registry
	if err := models.ValidateRegistry(&registry); err != nil {
		h.logger.Warn("Registry validation failed",
			"name", registry.Name,
			"error", err,
			"remote_addr", r.RemoteAddr)
		apierrors.WriteError(w, apierrors.ErrCodeValidationError, err.Error(), http.StatusBadRequest, nil)
		return
	}

	// Initialize packages map if nil
	if registry.Packages == nil {
		registry.Packages = make(map[string]*models.Package)
	}

	// Create registry (requires server admin, checked under the storage lock)
	var newAdmins *[]string
	if registry.Admins != nil {
		newAdmins = &registry.Admins
	}
	guard := h.authorizer.Guard(r, roles.CreateRegistry, roles.Target{Registry: registry.Name}, newAdmins)
	if err := h.store.CreateRegistry(r.Context(), &registry, guard); err != nil {
		var forbidden *storage.ForbiddenError
		if errors.As(err, &forbidden) {
			apierrors.WriteForbidden(w, forbidden)
			return
		}
		if err == storage.ErrAlreadyExists {
			code, msg, status := apierrors.MapStorageError(err, "registry")
			apierrors.WriteError(w, code, msg, status, nil)
			return
		}

		h.logger.Error("Failed to create registry",
			"name", registry.Name,
			"error", err)
		apierrors.WriteError(w, apierrors.ErrCodeStorageUnavailable, "Failed to create registry", http.StatusInternalServerError, nil)
		return
	}

	// Log successful creation
	h.logger.Info("Registry created",
		"name", registry.Name,
		"admin_count", len(registry.Admins),
		"custom_values", len(registry.CustomValues),
		"remote_addr", r.RemoteAddr)

	// Return created registry
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(registry)
}

// GetRegistry handles GET /api/v1/registry/:name
func (h *RegistryHandler) GetRegistry(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "name")

	// Get registry from storage
	registry, err := h.store.GetRegistry(r.Context(), registryName)
	if err != nil {
		if err == storage.ErrNotFound {
			code, msg, status := apierrors.MapStorageError(err, "registry")
			apierrors.WriteError(w, code, msg, status, nil)
			return
		}

		h.logger.Error("Failed to get registry",
			"registry", registryName,
			"error", err)
		apierrors.WriteError(w, apierrors.ErrCodeStorageUnavailable, "Failed to retrieve registry", http.StatusInternalServerError, nil)
		return
	}

	// Log retrieval
	h.logger.Debug("Registry retrieved",
		"registry", registryName,
		"package_count", len(registry.Packages))

	// Return registry
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(registry)
}

// UpdateRegistry handles PUT /api/v1/registry/:name
func (h *RegistryHandler) UpdateRegistry(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "name")

	// Parse request body: a partial update, absent fields keep their value
	var update models.RegistryUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		h.logger.Warn("Failed to decode registry update request",
			"registry", registryName,
			"error", err,
			"remote_addr", r.RemoteAddr)
		apierrors.WriteError(w, apierrors.ErrCodeValidationError, "Invalid JSON in request body", http.StatusBadRequest, nil)
		return
	}

	// Validate the fields present (and name, if given, against the URL)
	if err := update.Validate(registryName); err != nil {
		h.logger.Warn("Registry validation failed",
			"name", registryName,
			"error", err,
			"remote_addr", r.RemoteAddr)
		apierrors.WriteError(w, apierrors.ErrCodeValidationError, err.Error(), http.StatusBadRequest, nil)
		return
	}

	// Update registry (requires registry admin; checked and merged under the
	// storage lock, packages preserved)
	guard := h.authorizer.Guard(r, roles.UpdateRegistry, roles.Target{Registry: registryName}, update.Admins)
	registry, err := h.store.UpdateRegistry(r.Context(), registryName, &update, guard)
	if err != nil {
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

		h.logger.Error("Failed to update registry",
			"registry", registryName,
			"error", err)
		apierrors.WriteError(w, apierrors.ErrCodeStorageUnavailable, "Failed to update registry", http.StatusInternalServerError, nil)
		return
	}

	// Log successful update
	h.logger.Info("Registry updated",
		"name", registry.Name,
		"admin_count", len(registry.Admins),
		"custom_values", len(registry.CustomValues),
		"remote_addr", r.RemoteAddr)

	// Return updated registry
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(registry)
}

// DeleteRegistry handles DELETE /api/v1/registry/:name
func (h *RegistryHandler) DeleteRegistry(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "name")

	// Delete registry (requires server admin, checked under the storage
	// lock; cascade delete handled by storage layer)
	guard := h.authorizer.Guard(r, roles.DeleteRegistry, roles.Target{Registry: registryName}, nil)
	if err := h.store.DeleteRegistry(r.Context(), registryName, guard); err != nil {
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

		h.logger.Error("Failed to delete registry",
			"registry", registryName,
			"error", err)
		apierrors.WriteError(w, apierrors.ErrCodeStorageUnavailable, "Failed to delete registry", http.StatusInternalServerError, nil)
		return
	}

	// Log successful deletion
	h.logger.Info("Registry deleted",
		"name", registryName,
		"remote_addr", r.RemoteAddr)

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// ListRegistries handles GET /api/v1/registry
func (h *RegistryHandler) ListRegistries(w http.ResponseWriter, r *http.Request) {
	// Get all registries from storage
	registries, err := h.store.ListRegistries(r.Context())
	if err != nil {
		h.logger.Error("Failed to list registries",
			"error", err)
		apierrors.WriteError(w, apierrors.ErrCodeStorageUnavailable, "Failed to list registries", http.StatusInternalServerError, nil)
		return
	}

	// Log retrieval
	h.logger.Debug("Registries listed",
		"count", len(registries))

	// Return registries
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(registries)
}
