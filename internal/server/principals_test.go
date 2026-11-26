package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/criteo/command-launcher-registry/internal/server/servertest"
)

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// writeRoleList is one write that carries a role list: create or update of a
// registry (admins) or a package (maintainers). Each App is seeded with
// registry "repos" (admins ["root"]) and package "hotfix" (maintainers
// ["root"]).
type writeRoleList struct {
	name   string
	field  string
	method string
	path   string
	body   func(list []string) map[string]any
	// read returns the list stored for the written object, and whether it exists.
	read func(t *testing.T, app *servertest.App) ([]string, bool)
}

var roleListWrites = []writeRoleList{
	{
		name: "create registry", field: "admins", method: http.MethodPost, path: "/api/v1/registry",
		body: func(l []string) map[string]any { return map[string]any{"name": "tools", "admins": l} },
		read: func(t *testing.T, app *servertest.App) ([]string, bool) { return readRegistryAdmins(t, app, "tools") },
	},
	{
		name: "update registry", field: "admins", method: http.MethodPut, path: "/api/v1/registry/repos",
		body: func(l []string) map[string]any { return map[string]any{"name": "repos", "admins": l} },
		read: func(t *testing.T, app *servertest.App) ([]string, bool) { return readRegistryAdmins(t, app, "repos") },
	},
	{
		name: "create package", field: "maintainers", method: http.MethodPost, path: "/api/v1/registry/repos/package",
		body: func(l []string) map[string]any { return map[string]any{"name": "deploy", "maintainers": l} },
		read: func(t *testing.T, app *servertest.App) ([]string, bool) { return readMaintainers(t, app, "deploy") },
	},
	{
		name: "update package", field: "maintainers", method: http.MethodPut, path: "/api/v1/registry/repos/package/hotfix",
		body: func(l []string) map[string]any { return map[string]any{"name": "hotfix", "maintainers": l} },
		read: func(t *testing.T, app *servertest.App) ([]string, bool) { return readMaintainers(t, app, "hotfix") },
	},
}

// TestApp_RoleListElementValidation: an element of admins/maintainers that is
// empty or contains a comma or whitespace gives 400 naming the field and the
// element, and nothing is stored; valid lists are stored as sent.
func TestApp_RoleListElementValidation(t *testing.T) {
	invalid := []string{"", "a b", "a,b", "a\tb"}

	for _, w := range roleListWrites {
		for _, bad := range invalid {
			t.Run(fmt.Sprintf("%s rejects %q", w.name, bad), func(t *testing.T) {
				app := seededApp(t)
				before, existed := w.read(t, app)

				var body errorBody
				app.Do(servertest.User("alice"), w.method, w.path, w.body([]string{"alice", bad})).
					WantStatus(t, http.StatusBadRequest).
					JSON(t, &body)

				if body.Error.Code != "VALIDATION_ERROR" {
					t.Errorf("error code = %q, want VALIDATION_ERROR", body.Error.Code)
				}
				for _, want := range []string{w.field, fmt.Sprintf("%q", bad)} {
					if !strings.Contains(body.Error.Message, want) {
						t.Errorf("error message %q does not name %s", body.Error.Message, want)
					}
				}

				after, exists := w.read(t, app)
				if exists != existed || !slices.Equal(after, before) {
					t.Errorf("stored %s changed after 400: before %q (exists %v), after %q (exists %v)",
						w.field, before, existed, after, exists)
				}
			})
		}

		t.Run(w.name+" stores a valid list", func(t *testing.T) {
			app := seededApp(t)
			valid := []string{"alice", "team-build", "a.mazurov@criteo.com"}
			want := http.StatusOK
			if w.method == http.MethodPost {
				want = http.StatusCreated
			}
			app.Do(servertest.User("alice"), w.method, w.path, w.body(valid)).WantStatus(t, want)

			got, exists := w.read(t, app)
			if !exists || !slices.Equal(got, valid) {
				t.Errorf("stored %s = %q (exists %v), want %q", w.field, got, exists, valid)
			}
		})
	}
}

// TestApp_CreatePackageWithoutMaintainers: maintainers is optional on create.
func TestApp_CreatePackageWithoutMaintainers(t *testing.T) {
	app := seededApp(t)
	app.Do(servertest.User("alice"), http.MethodPost, "/api/v1/registry/repos/package",
		map[string]any{"name": "deploy"}).
		WantStatus(t, http.StatusCreated)

	got, exists := readMaintainers(t, app, "deploy")
	if !exists || len(got) != 0 {
		t.Errorf("maintainers = %q (exists %v), want none", got, exists)
	}
}

func seededApp(t *testing.T) *servertest.App {
	t.Helper()
	app := servertest.New(t)
	root := servertest.User("root")
	app.Do(root, http.MethodPost, "/api/v1/registry",
		map[string]any{"name": "repos", "admins": []string{"root"}}).
		WantStatus(t, http.StatusCreated)
	app.Do(root, http.MethodPost, "/api/v1/registry/repos/package",
		map[string]any{"name": "hotfix", "maintainers": []string{"root"}}).
		WantStatus(t, http.StatusCreated)
	return app
}

func readRegistryAdmins(t *testing.T, app *servertest.App, name string) ([]string, bool) {
	t.Helper()
	resp := app.Do(nil, http.MethodGet, "/api/v1/registry/"+name, nil)
	if resp.StatusCode == http.StatusNotFound {
		return nil, false
	}
	var reg models.Registry
	resp.WantStatus(t, http.StatusOK).JSON(t, &reg)
	return reg.Admins, true
}

func readMaintainers(t *testing.T, app *servertest.App, pkg string) ([]string, bool) {
	t.Helper()
	resp := app.Do(nil, http.MethodGet, "/api/v1/registry/repos/package/"+pkg, nil)
	if resp.StatusCode == http.StatusNotFound {
		return nil, false
	}
	var p models.Package
	resp.WantStatus(t, http.StatusOK).JSON(t, &p)
	return p.Maintainers, true
}
