package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/criteo/command-launcher-registry/internal/models"
)

var (
	// ErrNotFound is returned when a resource is not found
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists is returned when attempting to create a resource that already exists
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrStorageUnavailable is returned when storage operations fail
	ErrStorageUnavailable = errors.New("storage unavailable")

	// ErrImmutabilityViolation is returned when attempting to modify immutable data
	ErrImmutabilityViolation = errors.New("immutability violation")

	// ErrPartitionOverlap is returned when version partition ranges overlap
	ErrPartitionOverlap = errors.New("partition ranges overlap")
)

// Guard checks a write against the current data before the storage mutates
// it. The shared base storage calls it under its write lock, after the
// not-found checks and before the already-exists checks, so the decision and
// the mutation see the same data. A non-nil error denies the write: the
// storage returns it unchanged and persists nothing. A nil Guard allows.
//
// A Guard must not call the Store (the lock is held) and must not modify
// data.
type Guard func(data *models.Storage) error

// ForbiddenError is the error a Guard returns when the caller lacks the role
// a write requires. Role is the missing role and Target what it was checked
// on, for example "registry admin" and "repos".
type ForbiddenError struct {
	Role   string
	Target string
}

func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("requires %s on %s", e.Role, e.Target)
}

// Store defines the interface for storage operations. Every write operation
// takes a Guard (nil allows), checked under the storage lock.
type Store interface {
	// Registry operations
	CreateRegistry(ctx context.Context, r *models.Registry, guard Guard) error
	GetRegistry(ctx context.Context, name string) (*models.Registry, error)
	// UpdateRegistry applies a partial update to a registry under the storage
	// lock and returns the updated registry. Packages are preserved.
	UpdateRegistry(ctx context.Context, name string, u *models.RegistryUpdate, guard Guard) (*models.Registry, error)
	DeleteRegistry(ctx context.Context, name string, guard Guard) error
	ListRegistries(ctx context.Context) ([]*models.Registry, error)

	// Package operations
	CreatePackage(ctx context.Context, registryName string, p *models.Package, guard Guard) error
	GetPackage(ctx context.Context, registryName, packageName string) (*models.Package, error)
	// UpdatePackage applies a partial update to a package under the storage
	// lock and returns the updated package. Versions are preserved.
	UpdatePackage(ctx context.Context, registryName, packageName string, u *models.PackageUpdate, guard Guard) (*models.Package, error)
	DeletePackage(ctx context.Context, registryName, packageName string, guard Guard) error
	ListPackages(ctx context.Context, registryName string) ([]*models.Package, error)

	// Version operations
	CreateVersion(ctx context.Context, registryName, packageName string, v *models.Version, guard Guard) error
	GetVersion(ctx context.Context, registryName, packageName, version string) (*models.Version, error)
	DeleteVersion(ctx context.Context, registryName, packageName, version string, guard Guard) error
	ListVersions(ctx context.Context, registryName, packageName string) ([]*models.Version, error)

	// Index generation
	GetRegistryIndex(ctx context.Context, registryName string) ([]models.IndexEntry, error)

	// Close closes the storage
	Close() error
}
