package server_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/auth"
	"github.com/criteo/command-launcher-registry/internal/config"
	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/criteo/command-launcher-registry/internal/server"
	"github.com/criteo/command-launcher-registry/internal/server/servertest"
	"github.com/criteo/command-launcher-registry/internal/storage"
)

// Fixture identities for the registry role tests. The server admins are
// "root" and the group "g-server-admins"; registry "repos" has admins
// "repo-admin" and the group "g-repo-admins"; registry "other" has admin
// "other-admin".
var (
	serverAdminByName  = servertest.User("root")
	serverAdminByGroup = servertest.User("dave", "g-server-admins")
	repoAdmin          = servertest.User("repo-admin")
	repoAdminByGroup   = servertest.User("carol", "g-repo-admins")
	otherAdmin         = servertest.User("other-admin")
	noRole             = servertest.User("bob", "g-misc")
)

// newRolesApp starts an app with role enforcement on and the registries
// "repos" and "other" created by a server admin.
func newRolesApp(t *testing.T) *servertest.App {
	t.Helper()
	app := servertest.New(t, func(c *config.Config) {
		c.Admins = []string{"root", "g-server-admins"}
	})
	app.Do(serverAdminByName, http.MethodPost, "/api/v1/registry",
		map[string]any{"name": "repos", "description": "original", "admins": []string{"repo-admin", "g-repo-admins"}}).
		WantStatus(t, http.StatusCreated)
	app.Do(serverAdminByName, http.MethodPost, "/api/v1/registry",
		map[string]any{"name": "other", "admins": []string{"other-admin"}}).
		WantStatus(t, http.StatusCreated)
	return app
}

// registryWrite is one registry write action, with a check that reads the
// stored state back through the API and reports whether the write took
// effect.
type registryWrite struct {
	name    string
	method  string
	path    string
	body    any
	applied func(t *testing.T, app *servertest.App) bool
}

var registryWrites = []registryWrite{
	{
		name: "create", method: http.MethodPost, path: "/api/v1/registry",
		body: map[string]any{"name": "fresh"},
		applied: func(t *testing.T, app *servertest.App) bool {
			return app.Do(nil, http.MethodGet, "/api/v1/registry/fresh", nil).StatusCode == http.StatusOK
		},
	},
	{
		name: "update", method: http.MethodPut, path: "/api/v1/registry/repos",
		body: map[string]any{"description": "changed"},
		applied: func(t *testing.T, app *servertest.App) bool {
			var r models.Registry
			app.Do(nil, http.MethodGet, "/api/v1/registry/repos", nil).WantStatus(t, http.StatusOK).JSON(t, &r)
			return r.Description == "changed"
		},
	},
	{
		name: "delete", method: http.MethodDelete, path: "/api/v1/registry/repos",
		applied: func(t *testing.T, app *servertest.App) bool {
			return app.Do(nil, http.MethodGet, "/api/v1/registry/repos", nil).StatusCode == http.StatusNotFound
		},
	},
}

// TestApp_RegistryWriteRoles drives create, update and delete registry
// through the ADR 0001 matrix with enforcement on: create and delete need
// server admin, update needs registry admin of that registry.
func TestApp_RegistryWriteRoles(t *testing.T) {
	callers := []struct {
		name   string
		caller *auth.User
		want   map[string]int // by write name
	}{
		{"anonymous", nil, map[string]int{"create": 401, "update": 401, "delete": 401}},
		{"no role", noRole, map[string]int{"create": 403, "update": 403, "delete": 403}},
		{"registry admin of this registry", repoAdmin, map[string]int{"create": 403, "update": 200, "delete": 403}},
		{"registry admin of another registry", otherAdmin, map[string]int{"create": 403, "update": 403, "delete": 403}},
		{"registry admin via group", repoAdminByGroup, map[string]int{"create": 403, "update": 200, "delete": 403}},
		{"server admin by username", serverAdminByName, map[string]int{"create": 201, "update": 200, "delete": 204}},
		{"server admin by group", serverAdminByGroup, map[string]int{"create": 201, "update": 200, "delete": 204}},
	}
	for _, w := range registryWrites {
		for _, c := range callers {
			t.Run(w.name+"/"+c.name, func(t *testing.T) {
				app := newRolesApp(t)
				want := c.want[w.name]

				app.Do(c.caller, w.method, w.path, w.body).WantStatus(t, want)

				if got, wantApplied := w.applied(t, app), want < 300; got != wantApplied {
					t.Errorf("write applied = %v, want %v", got, wantApplied)
				}
			})
		}
	}
}

// TestApp_RegistryWritesEnforcementOff checks today's behaviour when roles
// are not enforced: any authenticated caller may create, update and delete
// registries; anonymous writes still get 401 (unless auth is "none").
func TestApp_RegistryWritesEnforcementOff(t *testing.T) {
	setups := []struct {
		name      string
		configure func(*config.Config)
		caller    *auth.User
		anonymous int
	}{
		{"empty server-admin list", func(c *config.Config) {}, noRole, http.StatusUnauthorized},
		{"auth none ignores the list", func(c *config.Config) {
			c.Auth.Type = "none"
			c.Admins = []string{"root"}
		}, nil, 0},
	}
	wantStatus := map[string]int{"create": 201, "update": 200, "delete": 204}
	for _, s := range setups {
		for _, w := range registryWrites {
			t.Run(s.name+"/"+w.name, func(t *testing.T) {
				app := servertest.New(t, s.configure)
				app.Do(s.caller, http.MethodPost, "/api/v1/registry",
					map[string]any{"name": "repos", "admins": []string{"someone-else"}}).
					WantStatus(t, http.StatusCreated)

				if s.anonymous != 0 {
					app.Do(nil, w.method, w.path, w.body).WantStatus(t, s.anonymous)
				}
				app.Do(s.caller, w.method, w.path, w.body).WantStatus(t, wantStatus[w.name])
				if !w.applied(t, app) {
					t.Errorf("write not applied")
				}
			})
		}
	}
}

// TestApp_RegistryAdminsManageCoAdmins checks that registry admins edit
// their own admins list (add and remove co-admins) and that a server admin
// creates a registry with admins set, which take effect at once.
func TestApp_RegistryAdminsManageCoAdmins(t *testing.T) {
	app := newRolesApp(t)
	erin := servertest.User("erin")

	app.Do(repoAdmin, http.MethodPut, "/api/v1/registry/repos",
		map[string]any{"admins": []string{"repo-admin", "erin"}}).WantStatus(t, http.StatusOK)
	app.Do(erin, http.MethodPut, "/api/v1/registry/repos",
		map[string]any{"description": "by erin"}).WantStatus(t, http.StatusOK)

	app.Do(repoAdmin, http.MethodPut, "/api/v1/registry/repos",
		map[string]any{"admins": []string{"repo-admin"}}).WantStatus(t, http.StatusOK)
	app.Do(erin, http.MethodPut, "/api/v1/registry/repos",
		map[string]any{"description": "by erin again"}).WantStatus(t, http.StatusForbidden)

	frank := servertest.User("frank")
	app.Do(serverAdminByName, http.MethodPost, "/api/v1/registry",
		map[string]any{"name": "team", "admins": []string{"frank"}}).WantStatus(t, http.StatusCreated)
	app.Do(frank, http.MethodPut, "/api/v1/registry/team",
		map[string]any{"description": "by frank"}).WantStatus(t, http.StatusOK)
	app.Do(frank, http.MethodDelete, "/api/v1/registry/team", nil).WantStatus(t, http.StatusForbidden)
}

// TestApp_ForbiddenBody checks the 403 envelope: code FORBIDDEN, a message
// naming the missing role and target, and details with both.
func TestApp_ForbiddenBody(t *testing.T) {
	tests := []struct {
		name         string
		method, path string
		body         any
		role, target string
	}{
		{"update", http.MethodPut, "/api/v1/registry/repos", map[string]any{"description": "x"}, "registry admin", "repos"},
		{"delete", http.MethodDelete, "/api/v1/registry/repos", nil, "server admin", "repos"},
		{"create", http.MethodPost, "/api/v1/registry", map[string]any{"name": "fresh"}, "server admin", "fresh"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newRolesApp(t)

			var body struct {
				Error struct {
					Code    string            `json:"code"`
					Message string            `json:"message"`
					Details map[string]string `json:"details"`
				} `json:"error"`
			}
			resp := app.Do(noRole, tt.method, tt.path, tt.body).WantStatus(t, http.StatusForbidden)
			resp.JSON(t, &body)

			if got := resp.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			if body.Error.Code != "FORBIDDEN" {
				t.Errorf("code = %q, want FORBIDDEN", body.Error.Code)
			}
			if want := "requires " + tt.role + " on " + tt.target; body.Error.Message != want {
				t.Errorf("message = %q, want %q", body.Error.Message, want)
			}
			wantDetails := map[string]string{"required_role": tt.role, "target": tt.target}
			if !reflect.DeepEqual(body.Error.Details, wantDetails) {
				t.Errorf("details = %v, want %v", body.Error.Details, wantDetails)
			}
		})
	}
}

// TestApp_RegistryWriteCheckOrder checks the order of the checks on a
// write: 401 → 400 → 404 → 403 → 409.
func TestApp_RegistryWriteCheckOrder(t *testing.T) {
	app := newRolesApp(t)

	// 401 before 400: anonymous with an invalid body.
	app.Do(nil, http.MethodPost, "/api/v1/registry", "{not json").WantStatus(t, http.StatusUnauthorized)
	// 400 before 403: no role, invalid body or invalid principal.
	app.Do(noRole, http.MethodPost, "/api/v1/registry", "{not json").WantStatus(t, http.StatusBadRequest)
	app.Do(noRole, http.MethodPut, "/api/v1/registry/repos",
		map[string]any{"admins": []string{"a b"}}).WantStatus(t, http.StatusBadRequest)
	// 404 before 403: reads are public, so existence is not secret.
	app.Do(noRole, http.MethodPut, "/api/v1/registry/missing",
		map[string]any{"description": "x"}).WantStatus(t, http.StatusNotFound)
	app.Do(noRole, http.MethodDelete, "/api/v1/registry/missing", nil).WantStatus(t, http.StatusNotFound)
	// 403 before 409: creating an existing registry without the role.
	app.Do(noRole, http.MethodPost, "/api/v1/registry", map[string]any{"name": "repos"}).
		WantStatus(t, http.StatusForbidden)
	app.Do(serverAdminByName, http.MethodPost, "/api/v1/registry", map[string]any{"name": "repos"}).
		WantStatus(t, http.StatusConflict)
}

// auditLines returns the audit lines the app has logged so far.
func auditLines(app *servertest.App) []map[string]any {
	var lines []map[string]any
	for _, line := range app.Logs() {
		if line["msg"] == "Write audit" {
			lines = append(lines, line)
		}
	}
	return lines
}

// wantAudit checks that line has every field in want (and only those of
// the optional fields that want names).
func wantAudit(t *testing.T, line map[string]any, want map[string]any) {
	t.Helper()
	for k, v := range want {
		if !reflect.DeepEqual(line[k], v) {
			t.Errorf("audit %s = %#v, want %#v (line %v)", k, line[k], v, line)
		}
	}
	for _, k := range []string{"matched_role", "missing_role", "old_admins", "new_admins"} {
		if _, wanted := want[k]; !wanted {
			if v, ok := line[k]; ok {
				t.Errorf("audit has %s = %#v, want none (line %v)", k, v, line)
			}
		}
	}
}

// TestApp_RegistryWriteAudit checks the audit line of allowed and denied
// writes, the old and new lists of an admins change, the line with
// enforcement off, and that no credentials are logged.
func TestApp_RegistryWriteAudit(t *testing.T) {
	t.Run("allowed and denied, enforcement on", func(t *testing.T) {
		app := newRolesApp(t)
		before := len(auditLines(app))

		app.Do(noRole, http.MethodPut, "/api/v1/registry/repos",
			map[string]any{"description": "x"}).WantStatus(t, http.StatusForbidden)
		app.Do(repoAdmin, http.MethodPut, "/api/v1/registry/repos",
			map[string]any{"admins": []string{"repo-admin", "erin"}}).WantStatus(t, http.StatusOK)
		app.Do(serverAdminByGroup, http.MethodDelete, "/api/v1/registry/other", nil).
			WantStatus(t, http.StatusNoContent)
		// Writes that stop before the decision are not audited.
		app.Do(nil, http.MethodPut, "/api/v1/registry/repos", map[string]any{"description": "x"}).
			WantStatus(t, http.StatusUnauthorized)
		app.Do(noRole, http.MethodDelete, "/api/v1/registry/missing", nil).WantStatus(t, http.StatusNotFound)

		lines := auditLines(app)[before:]
		if len(lines) != 3 {
			t.Fatalf("got %d audit lines, want 3: %v", len(lines), lines)
		}
		wantAudit(t, lines[0], map[string]any{
			"caller": "bob", "action": "update_registry", "target": "repos",
			"decision": "deny", "missing_role": "registry admin", "enforcement": "on",
		})
		wantAudit(t, lines[1], map[string]any{
			"caller": "repo-admin", "action": "update_registry", "target": "repos",
			"decision": "allow", "matched_role": "registry admin", "enforcement": "on",
			"old_admins": []any{"repo-admin", "g-repo-admins"},
			"new_admins": []any{"repo-admin", "erin"},
		})
		wantAudit(t, lines[2], map[string]any{
			"caller": "dave", "action": "delete_registry", "target": "other",
			"decision": "allow", "matched_role": "server admin", "enforcement": "on",
		})
	})

	t.Run("create with admins logs the new list", func(t *testing.T) {
		app := newRolesApp(t)
		lines := auditLines(app)
		if len(lines) != 2 {
			t.Fatalf("got %d audit lines for the two seeded creates, want 2: %v", len(lines), lines)
		}
		wantAudit(t, lines[0], map[string]any{
			"caller": "root", "action": "create_registry", "target": "repos",
			"decision": "allow", "matched_role": "server admin", "enforcement": "on",
			"old_admins": []any{}, "new_admins": []any{"repo-admin", "g-repo-admins"},
		})
	})

	t.Run("enforcement off", func(t *testing.T) {
		app := servertest.New(t)
		app.Do(noRole, http.MethodPost, "/api/v1/registry", map[string]any{"name": "repos"}).
			WantStatus(t, http.StatusCreated)

		lines := auditLines(app)
		if len(lines) != 1 {
			t.Fatalf("got %d audit lines, want 1: %v", len(lines), lines)
		}
		wantAudit(t, lines[0], map[string]any{
			"caller": "bob", "action": "create_registry", "target": "repos",
			"decision": "allow", "enforcement": "off",
		})
	})

	t.Run("no credentials in the logs", func(t *testing.T) {
		app := newRolesApp(t)
		app.Do(noRole, http.MethodPut, "/api/v1/registry/repos", map[string]any{"description": "x"})

		raw, err := json.Marshal(app.Logs())
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{servertest.Token(noRole), servertest.Token(serverAdminByName), "Bearer"} {
			if strings.Contains(string(raw), secret) {
				t.Errorf("logs contain %q: %s", secret, raw)
			}
		}
	})
}

// TestApp_RoleRemovedThenWriteDenied checks that a role removal takes
// effect on the very next write by that principal.
func TestApp_RoleRemovedThenWriteDenied(t *testing.T) {
	app := newRolesApp(t)

	app.Do(repoAdmin, http.MethodPut, "/api/v1/registry/repos",
		map[string]any{"description": "before"}).WantStatus(t, http.StatusOK)
	app.Do(serverAdminByName, http.MethodPut, "/api/v1/registry/repos",
		map[string]any{"admins": []string{"someone-else"}}).WantStatus(t, http.StatusOK)
	app.Do(repoAdmin, http.MethodPut, "/api/v1/registry/repos",
		map[string]any{"description": "after"}).WantStatus(t, http.StatusForbidden)
}

// staleReads is a Store whose read methods serve a stale copy of one
// registry and count every read call. The write methods reach the real
// storage, whose guard sees the current data under the lock.
type staleReads struct {
	storage.Store
	stale *models.Registry
	reads *atomic.Int32
}

func (s staleReads) GetRegistry(ctx context.Context, name string) (*models.Registry, error) {
	s.reads.Add(1)
	if name == s.stale.Name {
		return s.stale, nil
	}
	return s.Store.GetRegistry(ctx, name)
}

func (s staleReads) ListRegistries(ctx context.Context) ([]*models.Registry, error) {
	s.reads.Add(1)
	return s.Store.ListRegistries(ctx)
}

func (s staleReads) GetPackage(ctx context.Context, registryName, packageName string) (*models.Package, error) {
	s.reads.Add(1)
	return s.Store.GetPackage(ctx, registryName, packageName)
}

func (s staleReads) ListPackages(ctx context.Context, registryName string) ([]*models.Package, error) {
	s.reads.Add(1)
	return s.Store.ListPackages(ctx, registryName)
}

// TestApp_RoleCheckedUnderStorageLock checks that the role decision uses the
// data the storage holds under its write lock, not a separate read: with
// reads that still show the caller as registry admin after the role was
// removed, the write is denied, and the handler made no read call at all.
func TestApp_RoleCheckedUnderStorageLock(t *testing.T) {
	app := newRolesApp(t)
	app.Do(serverAdminByName, http.MethodPut, "/api/v1/registry/repos",
		map[string]any{"admins": []string{"someone-else"}}).WantStatus(t, http.StatusOK)

	stale := &models.Registry{Name: "repos", Description: "original", Admins: []string{"repo-admin"}, Packages: map[string]*models.Package{}}
	reads := &atomic.Int32{}
	app.Handler = server.NewHandler(app.Config, slog.New(slog.NewTextHandler(io.Discard, nil)), slog.New(slog.NewTextHandler(io.Discard, nil)),
		staleReads{Store: app.Store, stale: stale, reads: reads}, servertest.Authenticator{})

	// The stale read path is wired in: GET shows repo-admin as admin.
	var got models.Registry
	app.Do(nil, http.MethodGet, "/api/v1/registry/repos", nil).WantStatus(t, http.StatusOK).JSON(t, &got)
	if !reflect.DeepEqual(got.Admins, []string{"repo-admin"}) {
		t.Fatalf("stale read admins = %v, want [repo-admin]", got.Admins)
	}

	reads.Store(0)
	app.Do(repoAdmin, http.MethodPut, "/api/v1/registry/repos",
		map[string]any{"description": "changed"}).WantStatus(t, http.StatusForbidden)
	app.Do(repoAdmin, http.MethodDelete, "/api/v1/registry/repos", nil).WantStatus(t, http.StatusForbidden)
	if n := reads.Load(); n != 0 {
		t.Errorf("handlers made %d read calls during the writes, want 0 (no read-then-check)", n)
	}

	stored, err := app.Store.GetRegistry(context.Background(), "repos")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Description != "original" {
		t.Errorf("stored description = %q, want unchanged %q", stored.Description, "original")
	}
}
