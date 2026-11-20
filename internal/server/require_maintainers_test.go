package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/config"
	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/criteo/command-launcher-registry/internal/server/servertest"
)

// requireMaintainers turns the required-maintainers rule on.
func requireMaintainers(cfg *config.Config) { cfg.RequireMaintainers = true }

// newRequireMaintainersApp starts an app with the rule set to on, and creates
// registry "repos" and package "hotfix" maintained by alice and team-build.
// Roles are not enforced (no server admins), so only the rule can refuse.
func newRequireMaintainersApp(t *testing.T, on bool) *servertest.App {
	t.Helper()
	var app *servertest.App
	if on {
		app = servertest.New(t, requireMaintainers)
	} else {
		app = servertest.New(t)
	}
	alice := servertest.User("alice")
	app.Do(alice, http.MethodPost, "/api/v1/registry", map[string]any{"name": "repos"}).
		WantStatus(t, http.StatusCreated)
	app.Do(alice, http.MethodPost, "/api/v1/registry/repos/package", map[string]any{
		"name":        "hotfix",
		"maintainers": []string{"alice", "team-build"},
	}).WantStatus(t, http.StatusCreated)
	return app
}

func TestRequireMaintainers(t *testing.T) {
	tests := []struct {
		name            string
		on              bool
		method, path    string
		body            any
		wantStatus      int
		wantMaintainers []string // of hotfix afterwards, when wantStatus is 200
	}{
		{
			name: "on: create without maintainers", on: true,
			method: http.MethodPost, path: "/api/v1/registry/repos/package",
			body:       map[string]any{"name": "tool"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "on: create with null maintainers", on: true,
			method: http.MethodPost, path: "/api/v1/registry/repos/package",
			body:       map[string]any{"name": "tool", "maintainers": nil},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "on: create with empty maintainers", on: true,
			method: http.MethodPost, path: "/api/v1/registry/repos/package",
			body:       map[string]any{"name": "tool", "maintainers": []string{}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "on: create with maintainers", on: true,
			method: http.MethodPost, path: "/api/v1/registry/repos/package",
			body:       map[string]any{"name": "tool", "maintainers": []string{"bob"}},
			wantStatus: http.StatusCreated,
		},
		{
			name: "on: PUT sets maintainers to []", on: true,
			method: http.MethodPut, path: "/api/v1/registry/repos/package/hotfix",
			body:       map[string]any{"maintainers": []string{}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "on: PUT without maintainers keeps them", on: true,
			method: http.MethodPut, path: "/api/v1/registry/repos/package/hotfix",
			body:            map[string]any{"description": "New"},
			wantStatus:      http.StatusOK,
			wantMaintainers: []string{"alice", "team-build"},
		},
		{
			name: "on: PUT with null maintainers keeps them", on: true,
			method: http.MethodPut, path: "/api/v1/registry/repos/package/hotfix",
			body:            map[string]any{"maintainers": nil},
			wantStatus:      http.StatusOK,
			wantMaintainers: []string{"alice", "team-build"},
		},
		{
			name: "on: PUT replaces maintainers", on: true,
			method: http.MethodPut, path: "/api/v1/registry/repos/package/hotfix",
			body:            map[string]any{"maintainers": []string{"bob"}},
			wantStatus:      http.StatusOK,
			wantMaintainers: []string{"bob"},
		},
		{
			name: "off: create without maintainers", on: false,
			method: http.MethodPost, path: "/api/v1/registry/repos/package",
			body:       map[string]any{"name": "tool"},
			wantStatus: http.StatusCreated,
		},
		{
			name: "off: create with empty maintainers", on: false,
			method: http.MethodPost, path: "/api/v1/registry/repos/package",
			body:       map[string]any{"name": "tool", "maintainers": []string{}},
			wantStatus: http.StatusCreated,
		},
		{
			name: "off: PUT sets maintainers to []", on: false,
			method: http.MethodPut, path: "/api/v1/registry/repos/package/hotfix",
			body:            map[string]any{"maintainers": []string{}},
			wantStatus:      http.StatusOK,
			wantMaintainers: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newRequireMaintainersApp(t, tt.on)

			resp := app.Do(servertest.User("alice"), tt.method, tt.path, tt.body).WantStatus(t, tt.wantStatus)

			if tt.wantStatus == http.StatusBadRequest {
				var e struct {
					Error struct {
						Code    string `json:"code"`
						Message string `json:"message"`
					} `json:"error"`
				}
				resp.JSON(t, &e)
				if e.Error.Code != "VALIDATION_ERROR" || !strings.Contains(e.Error.Message, "requires at least one maintainer") {
					t.Errorf("error = %+v, want VALIDATION_ERROR saying the server requires at least one maintainer", e.Error)
				}
				// Nothing was written: no new package, hotfix unchanged.
				app.Do(nil, http.MethodGet, "/api/v1/registry/repos/package/tool", nil).WantStatus(t, http.StatusNotFound)
				assertEqual(t, "maintainers", getPackage(t, app).Maintainers, []string{"alice", "team-build"})
			}
			if tt.wantStatus == http.StatusOK {
				assertEqual(t, "maintainers", getPackage(t, app).Maintainers, tt.wantMaintainers)
			}
		})
	}
}

// TestRequireMaintainers_StoredEmptyPackage checks that the rule does not
// touch an existing package with empty maintainers: a PUT that omits them
// still works and the list stays empty.
func TestRequireMaintainers_StoredEmptyPackage(t *testing.T) {
	app := servertest.New(t, requireMaintainers)
	if err := app.Store.CreateRegistry(t.Context(), &models.Registry{Name: "repos", Packages: map[string]*models.Package{}}, nil); err != nil {
		t.Fatalf("seed registry: %v", err)
	}
	if err := app.Store.CreatePackage(t.Context(), "repos", &models.Package{Name: "hotfix", Versions: map[string]*models.Version{}}, nil); err != nil {
		t.Fatalf("seed package: %v", err)
	}

	app.Do(servertest.User("alice"), http.MethodPut, "/api/v1/registry/repos/package/hotfix",
		map[string]any{"description": "New"}).WantStatus(t, http.StatusOK)

	p := getPackage(t, app)
	assertEqual(t, "description", p.Description, "New")
	assertEqual(t, "maintainers", p.Maintainers, []string(nil))
}
