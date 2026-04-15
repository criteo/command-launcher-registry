package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/criteo/command-launcher-registry/internal/storage"
)

func TestVersionManifestLifecycle(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	storagePath := filepath.Join(t.TempDir(), "registry.json")
	store, err := storage.NewFileStorage(storagePath, "", logger)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, store.CreateRegistry(ctx, models.NewRegistry("tools", "", nil, nil)))
	require.NoError(t, store.CreatePackage(ctx, "tools", models.NewPackage("deploy", "", nil, nil)))

	handler := NewVersionHandler(store, logger)

	createBody := `{
		"name":"deploy",
		"version":"1.0.0",
		"checksum":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"url":"https://example.com/deploy-1.0.0.tgz",
		"startPartition":0,
		"endPartition":9,
		"manifest":"commands:\n  - name: deploy\n    description: Deploy the service\n"
	}`

	createReq := requestWithParams(http.MethodPost, "/api/v1/registry/tools/package/deploy/version", createBody,
		map[string]string{"name": "tools", "package": "deploy"})
	createRec := httptest.NewRecorder()
	handler.CreateVersion(createRec, createReq)

	require.Equal(t, http.StatusCreated, createRec.Code)
	assert.NotContains(t, createRec.Body.String(), "manifestDigest")

	getReq := requestWithParams(http.MethodGet, "/api/v1/registry/tools/package/deploy/version/1.0.0", "",
		map[string]string{"name": "tools", "package": "deploy", "version": "1.0.0"})
	getRec := httptest.NewRecorder()
	handler.GetVersion(getRec, getReq)

	require.Equal(t, http.StatusOK, getRec.Code)
	assert.NotContains(t, getRec.Body.String(), "manifestDigest")

	manifestReq := requestWithParams(http.MethodGet, "/api/v1/registry/tools/package/deploy/version/1.0.0/manifest", "",
		map[string]string{"name": "tools", "package": "deploy", "version": "1.0.0"})
	manifestRec := httptest.NewRecorder()
	handler.GetManifest(manifestRec, manifestReq)

	require.Equal(t, http.StatusOK, manifestRec.Code)
	assert.Equal(t, "application/json", manifestRec.Header().Get("Content-Type"))

	var canonical map[string]any
	require.NoError(t, json.Unmarshal(manifestRec.Body.Bytes(), &canonical))
	assert.Equal(t, map[string]any{
		"commands": []any{
			map[string]any{
				"description": "Deploy the service",
				"name":        "deploy",
			},
		},
	}, canonical)
}

func TestGetManifest_NotFoundWhenVersionHasNoManifest(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	storagePath := filepath.Join(t.TempDir(), "registry.json")
	store, err := storage.NewFileStorage(storagePath, "", logger)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, store.CreateRegistry(ctx, models.NewRegistry("tools", "", nil, nil)))
	require.NoError(t, store.CreatePackage(ctx, "tools", models.NewPackage("deploy", "", nil, nil)))
	require.NoError(t, store.CreateVersion(ctx, "tools", "deploy", models.NewVersion(
		"deploy",
		"1.0.0",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"https://example.com/deploy-1.0.0.tgz",
		0,
		9,
	)))

	handler := NewVersionHandler(store, logger)
	req := requestWithParams(http.MethodGet, "/api/v1/registry/tools/package/deploy/version/1.0.0/manifest", "",
		map[string]string{"name": "tools", "package": "deploy", "version": "1.0.0"})
	rec := httptest.NewRecorder()
	handler.GetManifest(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "MANIFEST_NOT_FOUND")
}

func requestWithParams(method, target, body string, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
