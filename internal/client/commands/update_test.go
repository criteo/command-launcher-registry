package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/config"
	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/criteo/command-launcher-registry/internal/server/servertest"
)

// updateServer runs the real server app (no auth) behind an httptest server,
// seeded with registry "repos" (admins, custom values) and package "hotfix"
// (maintainers, custom values). It records the PUT bodies the CLI sends.
type updateServer struct {
	app *servertest.App

	mu        sync.Mutex
	putBodies []map[string]any
}

func newUpdateServer(t *testing.T) *updateServer {
	t.Helper()
	s := &updateServer{app: servertest.New(t, func(c *config.Config) { c.Auth.Type = "none" })}

	s.app.Do(nil, http.MethodPost, "/api/v1/registry", map[string]any{
		"name":          "repos",
		"description":   "Repositories",
		"admins":        []string{"alice"},
		"custom_values": map[string]string{"owner": "build"},
	}).WantStatus(t, http.StatusCreated)
	s.app.Do(nil, http.MethodPost, "/api/v1/registry/repos/package", map[string]any{
		"name":          "hotfix",
		"description":   "Hotfix tool",
		"maintainers":   []string{"alice"},
		"custom_values": map[string]string{"tier": "1"},
	}).WantStatus(t, http.StatusCreated)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			data, _ := io.ReadAll(r.Body)
			var body map[string]any
			if err := json.Unmarshal(data, &body); err != nil {
				t.Errorf("PUT body is not a JSON object: %v: %s", err, data)
			}
			s.mu.Lock()
			s.putBodies = append(s.putBodies, body)
			s.mu.Unlock()
			r.Body = io.NopCloser(bytes.NewReader(data))
		}
		s.app.Handler.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	// Global flags are registered by Execute, which tests do not call: set
	// their variables directly. A token avoids reading the OS keychain.
	flagURL, flagToken = srv.URL, "user:password"
	t.Cleanup(func() { flagURL, flagToken = "", "" })
	return s
}

// run executes one CLI command line in-process. A failing command exits
// the test binary (the commands call os.Exit), which fails the test too.
func (s *updateServer) run(t *testing.T, args ...string) {
	t.Helper()
	resetUpdateFlags()
	defer resetUpdateFlags()
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("%v: %v", args, err)
	}
}

func (s *updateServer) lastPutBody(t *testing.T) map[string]any {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.putBodies) == 0 {
		t.Fatal("no PUT request was sent")
	}
	return s.putBodies[len(s.putBodies)-1]
}

func (s *updateServer) registry(t *testing.T) *models.Registry {
	t.Helper()
	r, err := s.app.Store.GetRegistry(context.Background(), "repos")
	if err != nil {
		t.Fatalf("get registry: %v", err)
	}
	return r
}

func (s *updateServer) pkg(t *testing.T) *models.Package {
	t.Helper()
	p, err := s.app.Store.GetPackage(context.Background(), "repos", "hotfix")
	if err != nil {
		t.Fatalf("get package: %v", err)
	}
	return p
}

// resetUpdateFlags clears the update flag variables, since the cobra
// commands are package globals shared by every test (other tests in this
// package leave them set).
func resetUpdateFlags() {
	regDescription, regAdmins, regCustomValues = "", nil, nil
	regClearAdmins, regClearCustomVal = false, false
	pkgDescription, pkgMaintainers, pkgCustomValues = "", nil, nil
	pkgClearMaint, pkgClearCustomVal = false, false
}

func wantEqual(t *testing.T, field string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %#v, want %#v", field, got, want)
	}
}

func TestRegistryUpdate_AgainstServer(t *testing.T) {
	s := newUpdateServer(t)

	s.run(t, "registry", "update", "repos", "--description", "New")
	if _, ok := s.lastPutBody(t)["name"]; ok {
		t.Errorf("registry update sent a name: %v", s.lastPutBody(t))
	}
	r := s.registry(t)
	wantEqual(t, "description", r.Description, "New")
	wantEqual(t, "admins", r.Admins, []string{"alice"})
	wantEqual(t, "custom_values", r.CustomValues, map[string]string{"owner": "build"})

	s.run(t, "registry", "update", "repos", "--admin", "bob", "--admin", "carol")
	wantEqual(t, "admins", s.registry(t).Admins, []string{"bob", "carol"})
	wantEqual(t, "description", s.registry(t).Description, "New")

	s.run(t, "registry", "update", "repos", "--clear-admins")
	wantEqual(t, "admins", s.registry(t).Admins, []string{})

	s.run(t, "registry", "update", "repos", "--clear-custom-values")
	wantEqual(t, "custom_values", s.registry(t).CustomValues, map[string]string{})
	wantEqual(t, "description", s.registry(t).Description, "New")
}

func TestPackageUpdate_AgainstServer(t *testing.T) {
	s := newUpdateServer(t)

	s.run(t, "package", "update", "repos", "hotfix", "--description", "New")
	if _, ok := s.lastPutBody(t)["name"]; ok {
		t.Errorf("package update sent a name: %v", s.lastPutBody(t))
	}
	p := s.pkg(t)
	wantEqual(t, "description", p.Description, "New")
	wantEqual(t, "maintainers", p.Maintainers, []string{"alice"})
	wantEqual(t, "custom_values", p.CustomValues, map[string]string{"tier": "1"})

	s.run(t, "package", "update", "repos", "hotfix", "--maintainer", "bob")
	wantEqual(t, "maintainers", s.pkg(t).Maintainers, []string{"bob"})

	s.run(t, "package", "update", "repos", "hotfix", "--clear-maintainers")
	wantEqual(t, "maintainers", s.pkg(t).Maintainers, []string{})

	s.run(t, "package", "update", "repos", "hotfix", "--clear-custom-values")
	wantEqual(t, "custom_values", s.pkg(t).CustomValues, map[string]string{})
	wantEqual(t, "description", s.pkg(t).Description, "New")
}
