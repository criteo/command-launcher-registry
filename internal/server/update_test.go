package server_test

import (
	"net/http"
	"reflect"
	"sync"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/criteo/command-launcher-registry/internal/server/servertest"
)

// seedUpdateFixture creates registry "repos" with admins and custom values,
// and package "hotfix" with maintainers, custom values and one version.
func seedUpdateFixture(t *testing.T, app *servertest.App) {
	t.Helper()
	alice := servertest.User("alice")
	app.Do(alice, http.MethodPost, "/api/v1/registry", map[string]any{
		"name":          "repos",
		"description":   "Repositories",
		"admins":        []string{"alice", "team-build"},
		"custom_values": map[string]string{"owner": "build"},
	}).WantStatus(t, http.StatusCreated)
	app.Do(alice, http.MethodPost, "/api/v1/registry/repos/package", map[string]any{
		"name":          "hotfix",
		"description":   "Hotfix tool",
		"maintainers":   []string{"alice", "team-build"},
		"custom_values": map[string]string{"tier": "1"},
	}).WantStatus(t, http.StatusCreated)
	app.Do(alice, http.MethodPost, "/api/v1/registry/repos/package/hotfix/version", map[string]any{
		"name":           "hotfix",
		"version":        "1.0.0",
		"checksum":       checksum,
		"url":            "https://example.com/hotfix-1.0.0.zip",
		"startPartition": 0,
		"endPartition":   9,
	}).WantStatus(t, http.StatusCreated)
}

func getRegistry(t *testing.T, app *servertest.App) models.Registry {
	t.Helper()
	var r models.Registry
	app.Do(nil, http.MethodGet, "/api/v1/registry/repos", nil).WantStatus(t, http.StatusOK).JSON(t, &r)
	return r
}

func getPackage(t *testing.T, app *servertest.App) models.Package {
	t.Helper()
	var p models.Package
	app.Do(nil, http.MethodGet, "/api/v1/registry/repos/package/hotfix", nil).WantStatus(t, http.StatusOK).JSON(t, &p)
	return p
}

func assertEqual(t *testing.T, field string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %#v, want %#v", field, got, want)
	}
}

func TestUpdateRegistry_PartialUpdate(t *testing.T) {
	tests := []struct {
		name             string
		body             any
		wantDescription  string
		wantAdmins       []string
		wantCustomValues map[string]string
	}{
		{
			name:             "only description keeps admins and custom values",
			body:             map[string]any{"description": "New"},
			wantDescription:  "New",
			wantAdmins:       []string{"alice", "team-build"},
			wantCustomValues: map[string]string{"owner": "build"},
		},
		{
			name:             "only admins replaces admins",
			body:             map[string]any{"admins": []string{"bob"}},
			wantDescription:  "Repositories",
			wantAdmins:       []string{"bob"},
			wantCustomValues: map[string]string{"owner": "build"},
		},
		{
			name:             "empty admins clears admins",
			body:             map[string]any{"admins": []string{}},
			wantDescription:  "Repositories",
			wantCustomValues: map[string]string{"owner": "build"},
		},
		{
			name:            "empty custom values clears custom values",
			body:            map[string]any{"custom_values": map[string]string{}},
			wantDescription: "Repositories",
			wantAdmins:      []string{"alice", "team-build"},
		},
		{
			name:             "empty description clears description",
			body:             map[string]any{"description": ""},
			wantAdmins:       []string{"alice", "team-build"},
			wantCustomValues: map[string]string{"owner": "build"},
		},
		{
			name:             "null fields keep the stored value",
			body:             `{"description": null, "admins": null, "custom_values": null}`,
			wantDescription:  "Repositories",
			wantAdmins:       []string{"alice", "team-build"},
			wantCustomValues: map[string]string{"owner": "build"},
		},
		{
			name:             "matching name is accepted",
			body:             map[string]any{"name": "repos", "description": "New"},
			wantDescription:  "New",
			wantAdmins:       []string{"alice", "team-build"},
			wantCustomValues: map[string]string{"owner": "build"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := servertest.New(t)
			seedUpdateFixture(t, app)

			var resp models.Registry
			app.Do(servertest.User("alice"), http.MethodPut, "/api/v1/registry/repos", tt.body).
				WantStatus(t, http.StatusOK).JSON(t, &resp)

			for _, r := range []models.Registry{resp, getRegistry(t, app)} {
				assertEqual(t, "description", r.Description, tt.wantDescription)
				assertEqual(t, "admins", r.Admins, tt.wantAdmins)
				assertEqual(t, "custom_values", r.CustomValues, tt.wantCustomValues)
			}
			if _, ok := getRegistry(t, app).Packages["hotfix"]; !ok {
				t.Errorf("package hotfix lost after registry update")
			}
		})
	}
}

func TestUpdatePackage_PartialUpdate(t *testing.T) {
	tests := []struct {
		name             string
		body             any
		wantDescription  string
		wantMaintainers  []string
		wantCustomValues map[string]string
	}{
		{
			name:             "only description keeps maintainers and custom values",
			body:             map[string]any{"description": "New"},
			wantDescription:  "New",
			wantMaintainers:  []string{"alice", "team-build"},
			wantCustomValues: map[string]string{"tier": "1"},
		},
		{
			name:             "only maintainers replaces maintainers",
			body:             map[string]any{"maintainers": []string{"bob"}},
			wantDescription:  "Hotfix tool",
			wantMaintainers:  []string{"bob"},
			wantCustomValues: map[string]string{"tier": "1"},
		},
		{
			name:             "empty maintainers clears maintainers",
			body:             map[string]any{"maintainers": []string{}},
			wantDescription:  "Hotfix tool",
			wantCustomValues: map[string]string{"tier": "1"},
		},
		{
			name:            "empty custom values clears custom values",
			body:            map[string]any{"custom_values": map[string]string{}},
			wantDescription: "Hotfix tool",
			wantMaintainers: []string{"alice", "team-build"},
		},
		{
			name:             "matching name is accepted",
			body:             map[string]any{"name": "hotfix", "description": "New"},
			wantDescription:  "New",
			wantMaintainers:  []string{"alice", "team-build"},
			wantCustomValues: map[string]string{"tier": "1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := servertest.New(t)
			seedUpdateFixture(t, app)

			var resp models.Package
			app.Do(servertest.User("alice"), http.MethodPut, "/api/v1/registry/repos/package/hotfix", tt.body).
				WantStatus(t, http.StatusOK).JSON(t, &resp)

			for _, p := range []models.Package{resp, getPackage(t, app)} {
				assertEqual(t, "description", p.Description, tt.wantDescription)
				assertEqual(t, "maintainers", p.Maintainers, tt.wantMaintainers)
				assertEqual(t, "custom_values", p.CustomValues, tt.wantCustomValues)
				if _, ok := p.Versions["1.0.0"]; !ok {
					t.Errorf("version 1.0.0 lost after package update")
				}
			}
		})
	}
}

func TestUpdate_Errors(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       any
		wantStatus int
	}{
		{"registry name differs from URL", "/api/v1/registry/repos", map[string]any{"name": "other"}, http.StatusBadRequest},
		{"package name differs from URL", "/api/v1/registry/repos/package/hotfix", map[string]any{"name": "other"}, http.StatusBadRequest},
		{"registry invalid JSON", "/api/v1/registry/repos", `{"admins": "alice"}`, http.StatusBadRequest},
		{"registry invalid custom value key", "/api/v1/registry/repos", map[string]any{"custom_values": map[string]string{"1bad": "x"}}, http.StatusBadRequest},
		{"unknown registry", "/api/v1/registry/missing", map[string]any{"description": "x"}, http.StatusNotFound},
		{"unknown package", "/api/v1/registry/repos/package/missing", map[string]any{"description": "x"}, http.StatusNotFound},
		{"package in unknown registry", "/api/v1/registry/missing/package/hotfix", map[string]any{"description": "x"}, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := servertest.New(t)
			seedUpdateFixture(t, app)

			app.Do(servertest.User("alice"), http.MethodPut, tt.path, tt.body).WantStatus(t, tt.wantStatus)

			// A rejected update changes nothing.
			r := getRegistry(t, app)
			assertEqual(t, "registry description", r.Description, "Repositories")
			assertEqual(t, "admins", r.Admins, []string{"alice", "team-build"})
			p := getPackage(t, app)
			assertEqual(t, "package description", p.Description, "Hotfix tool")
			assertEqual(t, "maintainers", p.Maintainers, []string{"alice", "team-build"})
		})
	}
}

// TestUpdate_ConcurrentFieldsNotLost sends concurrent PUTs that each change
// a different field. The merge runs under the storage lock, so every field
// ends with the value its own PUT sent.
func TestUpdate_ConcurrentFieldsNotLost(t *testing.T) {
	app := servertest.New(t)
	seedUpdateFixture(t, app)
	alice := servertest.User("alice")

	bodies := []struct {
		path string
		body map[string]any
	}{
		{"/api/v1/registry/repos", map[string]any{"description": "New"}},
		{"/api/v1/registry/repos", map[string]any{"admins": []string{"bob"}}},
		{"/api/v1/registry/repos", map[string]any{"custom_values": map[string]string{"owner": "ops"}}},
		{"/api/v1/registry/repos/package/hotfix", map[string]any{"description": "New"}},
		{"/api/v1/registry/repos/package/hotfix", map[string]any{"maintainers": []string{"bob"}}},
		{"/api/v1/registry/repos/package/hotfix", map[string]any{"custom_values": map[string]string{"tier": "2"}}},
	}
	const rounds = 10 // 60 requests, under the harness rate limit
	var wg sync.WaitGroup
	for range rounds {
		for _, b := range bodies {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if resp := app.Do(alice, http.MethodPut, b.path, b.body); resp.StatusCode != http.StatusOK {
					t.Errorf("PUT %s = %d: %s", b.path, resp.StatusCode, resp.Body)
				}
			}()
		}
	}
	wg.Wait()

	r := getRegistry(t, app)
	assertEqual(t, "registry description", r.Description, "New")
	assertEqual(t, "admins", r.Admins, []string{"bob"})
	assertEqual(t, "registry custom_values", r.CustomValues, map[string]string{"owner": "ops"})
	p := getPackage(t, app)
	assertEqual(t, "package description", p.Description, "New")
	assertEqual(t, "maintainers", p.Maintainers, []string{"bob"})
	assertEqual(t, "package custom_values", p.CustomValues, map[string]string{"tier": "2"})
	if _, ok := p.Versions["1.0.0"]; !ok {
		t.Errorf("version 1.0.0 lost")
	}
}
