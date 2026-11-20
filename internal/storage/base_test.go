package storage

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBaseStorage() *BaseStorage {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewBaseStorage(logger)
}

func TestBaseStorage_NewBaseStorage(t *testing.T) {
	bs := newTestBaseStorage()
	assert.NotNil(t, bs)
	assert.NotNil(t, bs.data)
	assert.NotNil(t, bs.data.Registries)
}

func TestBaseStorage_SetGetData(t *testing.T) {
	bs := newTestBaseStorage()

	// Create test data
	testData := &models.Storage{
		Registries: map[string]*models.Registry{
			"test-reg": {
				Name:     "test-reg",
				Packages: make(map[string]*models.Package),
			},
		},
	}

	// Set data
	bs.SetData(testData)

	// Get data
	data := bs.GetData()
	assert.Equal(t, 1, len(data.Registries))
	assert.Equal(t, "test-reg", data.Registries["test-reg"].Name)
}

func TestBaseStorage_MarshalUnmarshalData(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	// Create a registry
	reg := models.NewRegistry("test-reg", "Test Registry", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, nil)
	require.NoError(t, err)

	// Marshal
	jsonData, err := bs.MarshalData()
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "test-reg")

	// Create new storage and unmarshal
	bs2 := newTestBaseStorage()
	err = bs2.UnmarshalData(jsonData)
	require.NoError(t, err)

	// Verify data
	data := bs2.GetData()
	assert.Equal(t, 1, len(data.Registries))
	assert.Equal(t, "test-reg", data.Registries["test-reg"].Name)
}

func TestBaseStorage_CreateRegistry(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	reg := models.NewRegistry("test-reg", "Test Registry", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, nil)
	assert.NoError(t, err)

	// Verify created
	retrieved, err := bs.GetRegistry(ctx, "test-reg")
	require.NoError(t, err)
	assert.Equal(t, "test-reg", retrieved.Name)
	assert.Equal(t, "Test Registry", retrieved.Description)
}

func TestBaseStorage_CreateRegistry_AlreadyExists(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	reg := models.NewRegistry("test-reg", "Test Registry", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, nil)
	require.NoError(t, err)

	// Try to create again
	err = bs.CreateRegistry(ctx, reg, nil, nil)
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func TestBaseStorage_CreateRegistry_WithPersistCallback(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	persistCalled := false
	persistFunc := func() error {
		persistCalled = true
		return nil
	}

	reg := models.NewRegistry("test-reg", "Test Registry", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, persistFunc)
	assert.NoError(t, err)
	assert.True(t, persistCalled)
}

func TestBaseStorage_CreateRegistry_PersistFailure_Rollback(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	persistFunc := func() error {
		return assert.AnError
	}

	reg := models.NewRegistry("test-reg", "Test Registry", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, persistFunc)
	assert.ErrorIs(t, err, ErrStorageUnavailable)

	// Verify rollback - registry should not exist
	_, err = bs.GetRegistry(ctx, "test-reg")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestBaseStorage_GetRegistry_NotFound(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	_, err := bs.GetRegistry(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestBaseStorage_UpdateRegistry(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	// Create registry
	reg := models.NewRegistry("test-reg", "Test Registry", []string{"alice"}, nil)
	err := bs.CreateRegistry(ctx, reg, nil, nil)
	require.NoError(t, err)

	// Update the description only
	desc := "Updated Description"
	updated, err := bs.UpdateRegistry(ctx, "test-reg", &models.RegistryUpdate{Description: &desc}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Updated Description", updated.Description)

	// Verify update; omitted admins are kept
	retrieved, err := bs.GetRegistry(ctx, "test-reg")
	require.NoError(t, err)
	assert.Equal(t, "Updated Description", retrieved.Description)
	assert.Equal(t, []string{"alice"}, retrieved.Admins)

	// Unknown registry
	_, err = bs.UpdateRegistry(ctx, "missing", &models.RegistryUpdate{Description: &desc}, nil, nil)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestBaseStorage_UpdateRollsBackOnPersistFailure(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()
	failPersist := func() error { return assert.AnError }

	require.NoError(t, bs.CreateRegistry(ctx, models.NewRegistry("reg", "old", nil, nil), nil, nil))
	require.NoError(t, bs.CreatePackage(ctx, "reg", models.NewPackage("pkg", "old", nil, nil), nil, nil))

	desc := "new"
	_, err := bs.UpdateRegistry(ctx, "reg", &models.RegistryUpdate{Description: &desc}, nil, failPersist)
	assert.ErrorIs(t, err, ErrStorageUnavailable)
	_, err = bs.UpdatePackage(ctx, "reg", "pkg", &models.PackageUpdate{Description: &desc}, nil, failPersist)
	assert.ErrorIs(t, err, ErrStorageUnavailable)

	reg, err := bs.GetRegistry(ctx, "reg")
	require.NoError(t, err)
	assert.Equal(t, "old", reg.Description)
	pkg, err := bs.GetPackage(ctx, "reg", "pkg")
	require.NoError(t, err)
	assert.Equal(t, "old", pkg.Description)
}

func TestBaseStorage_DeleteRegistry(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	// Create registry
	reg := models.NewRegistry("test-reg", "Test Registry", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, nil)
	require.NoError(t, err)

	// Delete registry
	err = bs.DeleteRegistry(ctx, "test-reg", nil, nil)
	assert.NoError(t, err)

	// Verify deleted
	_, err = bs.GetRegistry(ctx, "test-reg")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestBaseStorage_ListRegistries(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	// Create registries
	for i := 0; i < 3; i++ {
		reg := models.NewRegistry(string(rune('a'+i)), "", nil, nil)
		err := bs.CreateRegistry(ctx, reg, nil, nil)
		require.NoError(t, err)
	}

	// List registries
	registries, err := bs.ListRegistries(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, len(registries))
}

func TestBaseStorage_CreatePackage(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	// Create registry first
	reg := models.NewRegistry("test-reg", "", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, nil)
	require.NoError(t, err)

	// Create package
	pkg := models.NewPackage("test-pkg", "Test Package", nil, nil)
	err = bs.CreatePackage(ctx, "test-reg", pkg, nil, nil)
	assert.NoError(t, err)

	// Verify created
	retrieved, err := bs.GetPackage(ctx, "test-reg", "test-pkg")
	require.NoError(t, err)
	assert.Equal(t, "test-pkg", retrieved.Name)
}

func TestBaseStorage_CreatePackage_RegistryNotFound(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	pkg := models.NewPackage("test-pkg", "", nil, nil)
	err := bs.CreatePackage(ctx, "nonexistent", pkg, nil, nil)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestBaseStorage_CreateVersion(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	// Setup registry and package
	reg := models.NewRegistry("test-reg", "", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, nil)
	require.NoError(t, err)

	pkg := models.NewPackage("test-pkg", "", nil, nil)
	err = bs.CreatePackage(ctx, "test-reg", pkg, nil, nil)
	require.NoError(t, err)

	// Create version
	ver := &models.Version{
		Name:           "test-pkg",
		Version:        "1.0.0",
		Checksum:       "abc123",
		URL:            "http://example.com/pkg.zip",
		StartPartition: 0,
		EndPartition:   9,
	}
	err = bs.CreateVersion(ctx, "test-reg", "test-pkg", ver, nil, nil)
	assert.NoError(t, err)

	// Verify created
	retrieved, err := bs.GetVersion(ctx, "test-reg", "test-pkg", "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", retrieved.Version)
}

func TestBaseStorage_CreateVersion_Immutability(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	// Setup
	reg := models.NewRegistry("test-reg", "", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, nil)
	require.NoError(t, err)

	pkg := models.NewPackage("test-pkg", "", nil, nil)
	err = bs.CreatePackage(ctx, "test-reg", pkg, nil, nil)
	require.NoError(t, err)

	ver := &models.Version{
		Name:           "test-pkg",
		Version:        "1.0.0",
		StartPartition: 0,
		EndPartition:   9,
	}
	err = bs.CreateVersion(ctx, "test-reg", "test-pkg", ver, nil, nil)
	require.NoError(t, err)

	// Try to create same version again
	err = bs.CreateVersion(ctx, "test-reg", "test-pkg", ver, nil, nil)
	assert.ErrorIs(t, err, ErrImmutabilityViolation)
}

func TestBaseStorage_CreateVersion_PartitionOverlapAllowed(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	// Setup
	reg := models.NewRegistry("test-reg", "", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, nil)
	require.NoError(t, err)

	pkg := models.NewPackage("test-pkg", "", nil, nil)
	err = bs.CreatePackage(ctx, "test-reg", pkg, nil, nil)
	require.NoError(t, err)

	// Create first version with partitions 0-4
	ver1 := &models.Version{
		Name:           "test-pkg",
		Version:        "1.0.0",
		StartPartition: 0,
		EndPartition:   4,
	}
	err = bs.CreateVersion(ctx, "test-reg", "test-pkg", ver1, nil, nil)
	require.NoError(t, err)

	// Create overlapping version with partitions 3-7 — should succeed
	ver2 := &models.Version{
		Name:           "test-pkg",
		Version:        "2.0.0",
		StartPartition: 3,
		EndPartition:   7,
	}
	err = bs.CreateVersion(ctx, "test-reg", "test-pkg", ver2, nil, nil)
	assert.NoError(t, err)
}

func TestBaseStorage_CreateVersion_StartGreaterThanEnd(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	// Setup
	reg := models.NewRegistry("test-reg", "", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, nil)
	require.NoError(t, err)

	pkg := models.NewPackage("test-pkg", "", nil, nil)
	err = bs.CreatePackage(ctx, "test-reg", pkg, nil, nil)
	require.NoError(t, err)

	// Create version where startPartition > endPartition (legacy convention)
	ver := &models.Version{
		Name:           "test-pkg",
		Version:        "1.0.0",
		StartPartition: 21,
		EndPartition:   9,
	}
	err = bs.CreateVersion(ctx, "test-reg", "test-pkg", ver, nil, nil)
	assert.NoError(t, err)
}

func TestBaseStorage_DeleteVersion(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	// Setup
	reg := models.NewRegistry("test-reg", "", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, nil)
	require.NoError(t, err)

	pkg := models.NewPackage("test-pkg", "", nil, nil)
	err = bs.CreatePackage(ctx, "test-reg", pkg, nil, nil)
	require.NoError(t, err)

	ver := &models.Version{
		Name:           "test-pkg",
		Version:        "1.0.0",
		StartPartition: 0,
		EndPartition:   9,
	}
	err = bs.CreateVersion(ctx, "test-reg", "test-pkg", ver, nil, nil)
	require.NoError(t, err)

	// Delete version
	err = bs.DeleteVersion(ctx, "test-reg", "test-pkg", "1.0.0", nil, nil)
	assert.NoError(t, err)

	// Verify deleted
	_, err = bs.GetVersion(ctx, "test-reg", "test-pkg", "1.0.0")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestBaseStorage_GetRegistryIndex(t *testing.T) {
	bs := newTestBaseStorage()
	ctx := context.Background()

	// Setup
	reg := models.NewRegistry("test-reg", "", nil, nil)
	err := bs.CreateRegistry(ctx, reg, nil, nil)
	require.NoError(t, err)

	pkg := models.NewPackage("test-pkg", "", nil, nil)
	err = bs.CreatePackage(ctx, "test-reg", pkg, nil, nil)
	require.NoError(t, err)

	ver := &models.Version{
		Name:           "test-pkg",
		Version:        "1.0.0",
		Checksum:       "abc123",
		URL:            "http://example.com/pkg.zip",
		StartPartition: 0,
		EndPartition:   9,
	}
	err = bs.CreateVersion(ctx, "test-reg", "test-pkg", ver, nil, nil)
	require.NoError(t, err)

	// Get index
	entries, err := bs.GetRegistryIndex(ctx, "test-reg")
	require.NoError(t, err)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, "test-pkg", entries[0].Name)
	assert.Equal(t, "1.0.0", entries[0].Version)
}

// TestBaseStorage_GuardDenies checks every write operation against a
// denying guard: the guard sees the current data, its error comes back
// unchanged, persist is not called and the data is not changed. It also
// checks that not-found comes before the guard and already-exists after it.
func TestBaseStorage_GuardDenies(t *testing.T) {
	ctx := context.Background()
	desc := "changed"
	ops := []struct {
		name string
		run  func(bs *BaseStorage, guard Guard, persist PersistFunc) error
	}{
		{"create registry (exists)", func(bs *BaseStorage, g Guard, p PersistFunc) error {
			return bs.CreateRegistry(ctx, models.NewRegistry("reg", "", nil, nil), g, p)
		}},
		{"update registry", func(bs *BaseStorage, g Guard, p PersistFunc) error {
			_, err := bs.UpdateRegistry(ctx, "reg", &models.RegistryUpdate{Description: &desc}, g, p)
			return err
		}},
		{"delete registry", func(bs *BaseStorage, g Guard, p PersistFunc) error {
			return bs.DeleteRegistry(ctx, "reg", g, p)
		}},
		{"create package (exists)", func(bs *BaseStorage, g Guard, p PersistFunc) error {
			return bs.CreatePackage(ctx, "reg", models.NewPackage("pkg", "", nil, nil), g, p)
		}},
		{"update package", func(bs *BaseStorage, g Guard, p PersistFunc) error {
			_, err := bs.UpdatePackage(ctx, "reg", "pkg", &models.PackageUpdate{Description: &desc}, g, p)
			return err
		}},
		{"delete package", func(bs *BaseStorage, g Guard, p PersistFunc) error {
			return bs.DeletePackage(ctx, "reg", "pkg", g, p)
		}},
		{"create version (exists)", func(bs *BaseStorage, g Guard, p PersistFunc) error {
			return bs.CreateVersion(ctx, "reg", "pkg", &models.Version{Name: "pkg", Version: "1.0.0"}, g, p)
		}},
		{"delete version", func(bs *BaseStorage, g Guard, p PersistFunc) error {
			return bs.DeleteVersion(ctx, "reg", "pkg", "1.0.0", g, p)
		}},
	}
	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			bs := newTestBaseStorage()
			require.NoError(t, bs.CreateRegistry(ctx, models.NewRegistry("reg", "orig", []string{"alice"}, nil), nil, nil))
			require.NoError(t, bs.CreatePackage(ctx, "reg", models.NewPackage("pkg", "orig", nil, nil), nil, nil))
			require.NoError(t, bs.CreateVersion(ctx, "reg", "pkg", &models.Version{Name: "pkg", Version: "1.0.0"}, nil, nil))
			before, err := bs.MarshalData()
			require.NoError(t, err)

			denied := &ForbiddenError{Role: "registry admin", Target: "reg"}
			var seen []string
			guard := func(data *models.Storage) error {
				seen = data.Registries["reg"].Admins
				return denied
			}
			persisted := false
			persist := func() error { persisted = true; return nil }

			err = op.run(bs, guard, persist)
			assert.Same(t, denied, err, "guard error returned unchanged")
			assert.Equal(t, []string{"alice"}, seen, "guard saw the current data")
			assert.False(t, persisted, "persist called after denial")
			after, err := bs.MarshalData()
			require.NoError(t, err)
			assert.JSONEq(t, string(before), string(after), "data changed after denial")

			// An allowing guard lets already-exists through for creates.
			if strings.HasSuffix(op.name, "(exists)") {
				err = op.run(bs, func(*models.Storage) error { return nil }, persist)
				assert.True(t, err == ErrAlreadyExists || err == ErrImmutabilityViolation, "got %v", err)
			}
		})
	}

	t.Run("not found before the guard", func(t *testing.T) {
		bs := newTestBaseStorage()
		called := false
		guard := func(*models.Storage) error { called = true; return &ForbiddenError{} }
		assert.Equal(t, ErrNotFound, bs.DeleteRegistry(ctx, "missing", guard, nil))
		assert.False(t, called, "guard called for a missing registry")
	})
}

func TestForbiddenError_Message(t *testing.T) {
	err := &ForbiddenError{Role: "registry admin", Target: "repos"}
	assert.Equal(t, "requires registry admin on repos", err.Error())
}
