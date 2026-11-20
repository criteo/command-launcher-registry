package server_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/auth"
	"github.com/criteo/command-launcher-registry/internal/config"
	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/criteo/command-launcher-registry/internal/server"
	"github.com/criteo/command-launcher-registry/internal/server/servertest"
)

// Fixture identities for the package role tests, on top of the registry
// fixtures: package "repos/hotfix" has maintainers "maint" and the group
// "g-hotfix-maint"; package "repos/tools" has maintainer "tools-maint".
var (
	maintainer        = servertest.User("maint")
	maintainerByGroup = servertest.User("erin", "g-hotfix-maint")
	otherMaintainer   = servertest.User("tools-maint")
)

// versionBody is a valid version body for package hotfix.
func versionBody(version string) map[string]any {
	return map[string]any{
		"name":           "hotfix",
		"version":        version,
		"checksum":       checksum,
		"url":            "https://example.com/hotfix-" + version + ".zip",
		"startPartition": 0,
		"endPartition":   9,
	}
}

// newPackageRolesApp starts newRolesApp and adds, as the registry admin,
// the packages "hotfix" (with version 1.0.0), "tools" and "orphan" (empty
// maintainers) to registry "repos".
func newPackageRolesApp(t *testing.T) *servertest.App {
	t.Helper()
	app := newRolesApp(t)
	for _, pkg := range []map[string]any{
		{"name": "hotfix", "maintainers": []string{"maint", "g-hotfix-maint"}},
		{"name": "tools", "maintainers": []string{"tools-maint"}},
		{"name": "orphan"},
	} {
		app.Do(repoAdmin, http.MethodPost, "/api/v1/registry/repos/package", pkg).
			WantStatus(t, http.StatusCreated)
	}
	app.Do(repoAdmin, http.MethodPost, "/api/v1/registry/repos/package/hotfix/version", versionBody("1.0.0")).
		WantStatus(t, http.StatusCreated)
	return app
}

// lookupPackage reads a package back through the API; ok is false when it
// does not exist.
func lookupPackage(t *testing.T, app *servertest.App, registry, name string) (pkg models.Package, ok bool) {
	t.Helper()
	resp := app.Do(nil, http.MethodGet, "/api/v1/registry/"+registry+"/package/"+name, nil)
	if resp.StatusCode == http.StatusNotFound {
		return pkg, false
	}
	resp.WantStatus(t, http.StatusOK).JSON(t, &pkg)
	return pkg, true
}

// packageWrites are the package and version write actions, each with a
// check that reports whether the write took effect.
var packageWrites = []registryWrite{
	{
		name: "create package", method: http.MethodPost, path: "/api/v1/registry/repos/package",
		body: map[string]any{"name": "fresh"},
		applied: func(t *testing.T, app *servertest.App) bool {
			_, ok := lookupPackage(t, app, "repos", "fresh")
			return ok
		},
	},
	{
		name: "update package", method: http.MethodPut, path: "/api/v1/registry/repos/package/hotfix",
		body: map[string]any{"description": "changed"},
		applied: func(t *testing.T, app *servertest.App) bool {
			pkg, _ := lookupPackage(t, app, "repos", "hotfix")
			return pkg.Description == "changed"
		},
	},
	{
		name: "delete package", method: http.MethodDelete, path: "/api/v1/registry/repos/package/hotfix",
		applied: func(t *testing.T, app *servertest.App) bool {
			_, ok := lookupPackage(t, app, "repos", "hotfix")
			return !ok
		},
	},
	{
		name: "create version", method: http.MethodPost, path: "/api/v1/registry/repos/package/hotfix/version",
		body: versionBody("2.0.0"),
		applied: func(t *testing.T, app *servertest.App) bool {
			return app.Do(nil, http.MethodGet, "/api/v1/registry/repos/package/hotfix/version/2.0.0", nil).
				StatusCode == http.StatusOK
		},
	},
	{
		name: "delete version", method: http.MethodDelete, path: "/api/v1/registry/repos/package/hotfix/version/1.0.0",
		applied: func(t *testing.T, app *servertest.App) bool {
			return app.Do(nil, http.MethodGet, "/api/v1/registry/repos/package/hotfix/version/1.0.0", nil).
				StatusCode == http.StatusNotFound
		},
	},
}

// TestApp_PackageWriteRoles drives the package and version writes through
// the ADR 0001 matrix with enforcement on: create and delete package need
// registry admin; update package and create and delete version need package
// maintainer, which registry admins and server admins hold.
func TestApp_PackageWriteRoles(t *testing.T) {
	all := func(status int) map[string]int {
		return map[string]int{"create package": status, "update package": status, "delete package": status,
			"create version": status, "delete version": status}
	}
	allowed := map[string]int{"create package": 201, "update package": 200, "delete package": 204,
		"create version": 201, "delete version": 204}
	maintainerWant := map[string]int{"create package": 403, "update package": 200, "delete package": 403,
		"create version": 201, "delete version": 204}

	callers := []struct {
		name   string
		caller *auth.User
		want   map[string]int // by write name
	}{
		{"anonymous", nil, all(401)},
		{"no role", noRole, all(403)},
		{"maintainer of this package", maintainer, maintainerWant},
		{"maintainer of another package", otherMaintainer, all(403)},
		{"maintainer via group", maintainerByGroup, maintainerWant},
		{"registry admin of this registry", repoAdmin, allowed},
		{"registry admin via group", repoAdminByGroup, allowed},
		{"registry admin of another registry", otherAdmin, all(403)},
		{"server admin by username", serverAdminByName, allowed},
		{"server admin by group", serverAdminByGroup, allowed},
	}
	for _, w := range packageWrites {
		for _, c := range callers {
			t.Run(w.name+"/"+c.name, func(t *testing.T) {
				app := newPackageRolesApp(t)
				want := c.want[w.name]

				app.Do(c.caller, w.method, w.path, w.body).WantStatus(t, want)

				if got, wantApplied := w.applied(t, app), want < 300; got != wantApplied {
					t.Errorf("write applied = %v, want %v", got, wantApplied)
				}
			})
		}
	}
}

// TestApp_PackageWritesEnforcementOff checks today's behaviour when roles
// are not enforced: any authenticated caller, maintainer or not, may do
// every package and version write; anonymous writes still get 401 (unless
// auth is "none").
func TestApp_PackageWritesEnforcementOff(t *testing.T) {
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
	wantStatus := map[string]int{"create package": 201, "update package": 200, "delete package": 204,
		"create version": 201, "delete version": 204}
	for _, s := range setups {
		for _, w := range packageWrites {
			t.Run(s.name+"/"+w.name, func(t *testing.T) {
				app := servertest.New(t, s.configure)
				app.Do(s.caller, http.MethodPost, "/api/v1/registry",
					map[string]any{"name": "repos", "admins": []string{"someone-else"}}).
					WantStatus(t, http.StatusCreated)
				app.Do(s.caller, http.MethodPost, "/api/v1/registry/repos/package",
					map[string]any{"name": "hotfix", "maintainers": []string{"someone-else"}}).
					WantStatus(t, http.StatusCreated)
				app.Do(s.caller, http.MethodPost, "/api/v1/registry/repos/package/hotfix/version", versionBody("1.0.0")).
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

// TestApp_MaintainersManageCoMaintainers checks that a package maintainer
// adds and removes co-maintainers and publishes a version, but cannot
// create or delete packages.
func TestApp_MaintainersManageCoMaintainers(t *testing.T) {
	app := newPackageRolesApp(t)
	frank := servertest.User("frank")

	app.Do(maintainer, http.MethodPut, "/api/v1/registry/repos/package/hotfix",
		map[string]any{"maintainers": []string{"maint", "frank"}}).WantStatus(t, http.StatusOK)
	app.Do(frank, http.MethodPost, "/api/v1/registry/repos/package/hotfix/version", versionBody("1.1.0")).
		WantStatus(t, http.StatusCreated)

	app.Do(maintainer, http.MethodPut, "/api/v1/registry/repos/package/hotfix",
		map[string]any{"maintainers": []string{"maint"}}).WantStatus(t, http.StatusOK)
	app.Do(frank, http.MethodPost, "/api/v1/registry/repos/package/hotfix/version", versionBody("1.2.0")).
		WantStatus(t, http.StatusForbidden)

	app.Do(maintainer, http.MethodPost, "/api/v1/registry/repos/package/hotfix/version", versionBody("1.2.0")).
		WantStatus(t, http.StatusCreated)
	app.Do(maintainer, http.MethodPost, "/api/v1/registry/repos/package",
		map[string]any{"name": "mine", "maintainers": []string{"maint"}}).WantStatus(t, http.StatusForbidden)
	app.Do(maintainer, http.MethodDelete, "/api/v1/registry/repos/package/hotfix", nil).
		WantStatus(t, http.StatusForbidden)

	// A maintainer of one package has no rights on another.
	app.Do(maintainer, http.MethodPut, "/api/v1/registry/repos/package/tools",
		map[string]any{"description": "x"}).WantStatus(t, http.StatusForbidden)
}

// TestApp_PackageWithoutMaintainers checks that a package with an empty
// maintainers list can be changed only by its registry admins and server
// admins.
func TestApp_PackageWithoutMaintainers(t *testing.T) {
	app := newPackageRolesApp(t)
	path := "/api/v1/registry/repos/package/orphan"

	for _, c := range []*auth.User{noRole, maintainer, otherMaintainer, otherAdmin} {
		app.Do(c, http.MethodPut, path, map[string]any{"description": "x"}).WantStatus(t, http.StatusForbidden)
		app.Do(c, http.MethodPost, path+"/version", versionBody("1.0.0")).WantStatus(t, http.StatusForbidden)
	}
	app.Do(repoAdmin, http.MethodPut, path, map[string]any{"description": "by repo admin"}).
		WantStatus(t, http.StatusOK)
	app.Do(repoAdmin, http.MethodPost, path+"/version", versionBody("1.0.0")).WantStatus(t, http.StatusCreated)
	app.Do(serverAdminByName, http.MethodPost, path+"/version", versionBody("2.0.0")).
		WantStatus(t, http.StatusCreated)
}

// TestApp_CreatePackageKeepsMaintainers checks that creating a package does
// not add the caller to maintainers, with or without a list in the body.
func TestApp_CreatePackageKeepsMaintainers(t *testing.T) {
	app := newRolesApp(t)

	app.Do(repoAdmin, http.MethodPost, "/api/v1/registry/repos/package",
		map[string]any{"name": "bare"}).WantStatus(t, http.StatusCreated)
	app.Do(repoAdmin, http.MethodPost, "/api/v1/registry/repos/package",
		map[string]any{"name": "listed", "maintainers": []string{"alice"}}).WantStatus(t, http.StatusCreated)

	if pkg, _ := lookupPackage(t, app, "repos", "bare"); len(pkg.Maintainers) != 0 {
		t.Errorf("bare maintainers = %v, want none", pkg.Maintainers)
	}
	if pkg, _ := lookupPackage(t, app, "repos", "listed"); !reflect.DeepEqual(pkg.Maintainers, []string{"alice"}) {
		t.Errorf("listed maintainers = %v, want [alice]", pkg.Maintainers)
	}
}

// TestApp_PackageForbiddenBody checks the 403 message of package and
// version writes: registry admin is checked on the registry, package
// maintainer on the package.
func TestApp_PackageForbiddenBody(t *testing.T) {
	scopes := map[string][2]string{
		"create package": {"registry admin", "repos"},
		"update package": {"package maintainer", "repos/hotfix"},
		"delete package": {"registry admin", "repos"},
		"create version": {"package maintainer", "repos/hotfix"},
		"delete version": {"package maintainer", "repos/hotfix"},
	}
	for _, w := range packageWrites {
		t.Run(w.name, func(t *testing.T) {
			app := newPackageRolesApp(t)

			var body struct {
				Error struct {
					Code    string            `json:"code"`
					Message string            `json:"message"`
					Details map[string]string `json:"details"`
				} `json:"error"`
			}
			app.Do(noRole, w.method, w.path, w.body).WantStatus(t, http.StatusForbidden).JSON(t, &body)

			role, target := scopes[w.name][0], scopes[w.name][1]
			if body.Error.Code != "FORBIDDEN" {
				t.Errorf("code = %q, want FORBIDDEN", body.Error.Code)
			}
			if want := "requires " + role + " on " + target; body.Error.Message != want {
				t.Errorf("message = %q, want %q", body.Error.Message, want)
			}
			wantDetails := map[string]string{"required_role": role, "target": target}
			if !reflect.DeepEqual(body.Error.Details, wantDetails) {
				t.Errorf("details = %v, want %v", body.Error.Details, wantDetails)
			}
		})
	}
}

// TestApp_PackageWriteCheckOrder checks the order of the checks on package
// and version writes: 401 → 400 → 404 → 403 → 409.
func TestApp_PackageWriteCheckOrder(t *testing.T) {
	app := newPackageRolesApp(t)
	pkgs := "/api/v1/registry/repos/package"

	// 401 before 400.
	app.Do(nil, http.MethodPost, pkgs, "{not json").WantStatus(t, http.StatusUnauthorized)
	// 400 before 403: invalid body, invalid principal, invalid partitions.
	app.Do(noRole, http.MethodPost, pkgs, "{not json").WantStatus(t, http.StatusBadRequest)
	app.Do(noRole, http.MethodPut, pkgs+"/hotfix",
		map[string]any{"maintainers": []string{"a b"}}).WantStatus(t, http.StatusBadRequest)
	bad := versionBody("3.0.0")
	bad["startPartition"] = -1
	app.Do(noRole, http.MethodPost, pkgs+"/hotfix/version", bad).WantStatus(t, http.StatusBadRequest)
	// 404 before 403.
	app.Do(noRole, http.MethodPost, "/api/v1/registry/missing/package",
		map[string]any{"name": "x"}).WantStatus(t, http.StatusNotFound)
	app.Do(noRole, http.MethodPut, pkgs+"/missing", map[string]any{"description": "x"}).
		WantStatus(t, http.StatusNotFound)
	app.Do(noRole, http.MethodDelete, pkgs+"/missing", nil).WantStatus(t, http.StatusNotFound)
	app.Do(noRole, http.MethodPost, pkgs+"/missing/version", versionBody("1.0.0")).
		WantStatus(t, http.StatusNotFound)
	app.Do(noRole, http.MethodDelete, pkgs+"/hotfix/version/9.9.9", nil).WantStatus(t, http.StatusNotFound)
	// 403 before 409: an existing package or version without the role.
	app.Do(noRole, http.MethodPost, pkgs, map[string]any{"name": "hotfix"}).WantStatus(t, http.StatusForbidden)
	app.Do(repoAdmin, http.MethodPost, pkgs, map[string]any{"name": "hotfix"}).WantStatus(t, http.StatusConflict)
	app.Do(noRole, http.MethodPost, pkgs+"/hotfix/version", versionBody("1.0.0")).
		WantStatus(t, http.StatusForbidden)
	app.Do(maintainer, http.MethodPost, pkgs+"/hotfix/version", versionBody("1.0.0")).
		WantStatus(t, http.StatusConflict)
}

// TestApp_PackageWriteAudit checks the audit lines of package and version
// writes, and that a maintainers change logs the old and new lists.
func TestApp_PackageWriteAudit(t *testing.T) {
	app := newPackageRolesApp(t)

	t.Run("create logs the maintainers the package starts with", func(t *testing.T) {
		lines := auditLines(app)
		// 2 registry creates, then 3 package creates and 1 version.
		if len(lines) != 6 {
			t.Fatalf("got %d audit lines, want 6: %v", len(lines), lines)
		}
		wantPackageAudit(t, lines[2], map[string]any{
			"caller": "repo-admin", "action": "create_package", "target": "repos/hotfix",
			"decision": "allow", "matched_role": "registry admin", "enforcement": "on",
			"old_maintainers": []any{}, "new_maintainers": []any{"maint", "g-hotfix-maint"},
		})
		wantPackageAudit(t, lines[4], map[string]any{
			"caller": "repo-admin", "action": "create_package", "target": "repos/orphan",
			"decision": "allow", "matched_role": "registry admin", "enforcement": "on",
			"old_maintainers": []any{}, "new_maintainers": []any{},
		})
		wantPackageAudit(t, lines[5], map[string]any{
			"caller": "repo-admin", "action": "create_version", "target": "repos/hotfix/1.0.0",
			"decision": "allow", "matched_role": "registry admin", "enforcement": "on",
		})
	})

	t.Run("maintainers change, denied writes, version delete", func(t *testing.T) {
		before := len(auditLines(app))

		app.Do(maintainerByGroup, http.MethodPut, "/api/v1/registry/repos/package/hotfix",
			map[string]any{"maintainers": []string{"maint", "frank"}}).WantStatus(t, http.StatusOK)
		app.Do(maintainer, http.MethodPut, "/api/v1/registry/repos/package/hotfix",
			map[string]any{"description": "no list change"}).WantStatus(t, http.StatusOK)
		app.Do(otherMaintainer, http.MethodPut, "/api/v1/registry/repos/package/hotfix",
			map[string]any{"maintainers": []string{"tools-maint"}}).WantStatus(t, http.StatusForbidden)
		app.Do(maintainer, http.MethodDelete, "/api/v1/registry/repos/package/hotfix", nil).
			WantStatus(t, http.StatusForbidden)
		app.Do(maintainer, http.MethodDelete, "/api/v1/registry/repos/package/hotfix/version/1.0.0", nil).
			WantStatus(t, http.StatusNoContent)

		lines := auditLines(app)[before:]
		if len(lines) != 5 {
			t.Fatalf("got %d audit lines, want 5: %v", len(lines), lines)
		}
		wantPackageAudit(t, lines[0], map[string]any{
			"caller": "erin", "action": "update_package", "target": "repos/hotfix",
			"decision": "allow", "matched_role": "package maintainer", "enforcement": "on",
			"old_maintainers": []any{"maint", "g-hotfix-maint"}, "new_maintainers": []any{"maint", "frank"},
		})
		wantPackageAudit(t, lines[1], map[string]any{
			"caller": "maint", "action": "update_package", "target": "repos/hotfix",
			"decision": "allow", "matched_role": "package maintainer", "enforcement": "on",
		})
		wantPackageAudit(t, lines[2], map[string]any{
			"caller": "tools-maint", "action": "update_package", "target": "repos/hotfix",
			"decision": "deny", "missing_role": "package maintainer", "enforcement": "on",
			"old_maintainers": []any{"maint", "frank"}, "new_maintainers": []any{"tools-maint"},
		})
		wantPackageAudit(t, lines[3], map[string]any{
			"caller": "maint", "action": "delete_package", "target": "repos/hotfix",
			"decision": "deny", "missing_role": "registry admin", "enforcement": "on",
		})
		wantPackageAudit(t, lines[4], map[string]any{
			"caller": "maint", "action": "delete_version", "target": "repos/hotfix/1.0.0",
			"decision": "allow", "matched_role": "package maintainer", "enforcement": "on",
		})
	})

	t.Run("enforcement off", func(t *testing.T) {
		app := servertest.New(t)
		app.Do(noRole, http.MethodPost, "/api/v1/registry", map[string]any{"name": "repos"}).
			WantStatus(t, http.StatusCreated)
		app.Do(noRole, http.MethodPost, "/api/v1/registry/repos/package", map[string]any{"name": "hotfix"}).
			WantStatus(t, http.StatusCreated)

		lines := auditLines(app)
		if len(lines) != 2 {
			t.Fatalf("got %d audit lines, want 2: %v", len(lines), lines)
		}
		wantPackageAudit(t, lines[1], map[string]any{
			"caller": "bob", "action": "create_package", "target": "repos/hotfix",
			"decision": "allow", "enforcement": "off",
			"old_maintainers": []any{}, "new_maintainers": []any{},
		})
	})
}

// wantPackageAudit is wantAudit for package lines: it also checks that the
// maintainers lists appear only when want names them, and that no admins
// lists appear.
func wantPackageAudit(t *testing.T, line map[string]any, want map[string]any) {
	t.Helper()
	wantAudit(t, line, want)
	for _, k := range []string{"old_maintainers", "new_maintainers"} {
		if _, wanted := want[k]; !wanted {
			if v, ok := line[k]; ok {
				t.Errorf("audit has %s = %#v, want none (line %v)", k, v, line)
			}
		}
	}
}

// TestApp_IndexUnchangedByRoles checks that index.json, served with
// enforcement on, holds only the Command Launcher fields: no maintainers,
// admins or other role data.
func TestApp_IndexUnchangedByRoles(t *testing.T) {
	app := newPackageRolesApp(t)

	var raw []map[string]any
	app.Do(nil, http.MethodGet, "/api/v1/registry/repos/index.json", nil).WantStatus(t, http.StatusOK).JSON(t, &raw)
	want := []map[string]any{{
		"name":           "hotfix",
		"version":        "1.0.0",
		"checksum":       checksum,
		"url":            "https://example.com/hotfix-1.0.0.zip",
		"startPartition": float64(0),
		"endPartition":   float64(9),
	}}
	if !reflect.DeepEqual(raw, want) {
		got, _ := json.Marshal(raw)
		t.Errorf("index.json = %s, want only the Command Launcher fields of hotfix 1.0.0", got)
	}
}

// TestApp_PackageRoleCheckedUnderStorageLock checks that package and version
// writes decide on the data under the storage lock: with reads that still
// show the caller as maintainer after it was removed, the writes are
// denied, and the handlers made no read call.
func TestApp_PackageRoleCheckedUnderStorageLock(t *testing.T) {
	app := newPackageRolesApp(t)
	app.Do(repoAdmin, http.MethodPut, "/api/v1/registry/repos/package/hotfix",
		map[string]any{"maintainers": []string{"someone-else"}}).WantStatus(t, http.StatusOK)

	stale := &models.Registry{Name: "repos", Admins: []string{"repo-admin"}, Packages: map[string]*models.Package{
		"hotfix": {Name: "hotfix", Maintainers: []string{"maint"}, Versions: map[string]*models.Version{}},
	}}
	reads := &atomic.Int32{}
	app.Handler = server.NewHandler(app.Config, slog.New(slog.NewTextHandler(io.Discard, nil)),
		staleReads{Store: app.Store, stale: stale, reads: reads}, servertest.Authenticator{})

	app.Do(maintainer, http.MethodPut, "/api/v1/registry/repos/package/hotfix",
		map[string]any{"description": "changed"}).WantStatus(t, http.StatusForbidden)
	app.Do(maintainer, http.MethodPost, "/api/v1/registry/repos/package/hotfix/version", versionBody("2.0.0")).
		WantStatus(t, http.StatusForbidden)
	app.Do(maintainer, http.MethodDelete, "/api/v1/registry/repos/package/hotfix/version/1.0.0", nil).
		WantStatus(t, http.StatusForbidden)
	if n := reads.Load(); n != 0 {
		t.Errorf("handlers made %d read calls during the writes, want 0 (no read-then-check)", n)
	}

	stored, err := app.Store.GetPackage(context.Background(), "repos", "hotfix")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Description != "" || len(stored.Versions) != 1 {
		t.Errorf("stored package changed: description %q, %d versions", stored.Description, len(stored.Versions))
	}
}
