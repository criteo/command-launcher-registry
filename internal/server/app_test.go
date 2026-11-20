package server_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/config"
	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/criteo/command-launcher-registry/internal/server/servertest"
)

const checksum = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

// TestApp_TodaysBehaviour pins the API behaviour before role enforcement:
// reads are public, writes need credentials, and any authenticated caller
// can create a registry, a package and a version.
func TestApp_TodaysBehaviour(t *testing.T) {
	app := servertest.New(t)
	alice := servertest.User("alice", "team-build")

	// Anonymous reads work.
	app.Do(nil, http.MethodGet, "/api/v1/health", nil).WantStatus(t, http.StatusOK)
	app.Do(nil, http.MethodGet, "/api/v1/registry", nil).WantStatus(t, http.StatusOK)

	// Anonymous writes get 401.
	app.Do(nil, http.MethodPost, "/api/v1/registry", map[string]any{"name": "repos"}).
		WantStatus(t, http.StatusUnauthorized)

	// An authenticated caller creates registry -> package -> version.
	app.Do(alice, http.MethodPost, "/api/v1/registry",
		map[string]any{"name": "repos", "description": "Repositories"}).
		WantStatus(t, http.StatusCreated)
	app.Do(alice, http.MethodPost, "/api/v1/registry/repos/package",
		map[string]any{"name": "hotfix", "maintainers": []string{"alice"}}).
		WantStatus(t, http.StatusCreated)
	app.Do(alice, http.MethodPost, "/api/v1/registry/repos/package/hotfix/version",
		map[string]any{
			"name":           "hotfix",
			"version":        "1.0.0",
			"checksum":       checksum,
			"url":            "https://example.com/hotfix-1.0.0.zip",
			"startPartition": 0,
			"endPartition":   9,
		}).
		WantStatus(t, http.StatusCreated)

	// The anonymous index.json lists the version.
	var index []models.IndexEntry
	app.Do(nil, http.MethodGet, "/api/v1/registry/repos/index.json", nil).
		WantStatus(t, http.StatusOK).
		JSON(t, &index)
	want := []models.IndexEntry{{
		Name:           "hotfix",
		Version:        "1.0.0",
		Checksum:       checksum,
		URL:            "https://example.com/hotfix-1.0.0.zip",
		StartPartition: 0,
		EndPartition:   9,
	}}
	if !reflect.DeepEqual(index, want) {
		t.Errorf("index.json = %+v, want %+v", index, want)
	}

	// The anonymous package read shows the stored maintainers.
	var pkg models.Package
	app.Do(nil, http.MethodGet, "/api/v1/registry/repos/package/hotfix", nil).
		WantStatus(t, http.StatusOK).
		JSON(t, &pkg)
	if !reflect.DeepEqual(pkg.Maintainers, []string{"alice"}) {
		t.Errorf("maintainers = %v, want [alice]", pkg.Maintainers)
	}
}

func TestApp_TestAuthenticator(t *testing.T) {
	app := servertest.New(t)

	var who struct {
		Username string   `json:"username"`
		Groups   []string `json:"groups"`
	}
	app.Do(servertest.User("bob", "team-a", "team-b"), http.MethodGet, "/api/v1/whoami", nil).
		WantStatus(t, http.StatusOK).
		JSON(t, &who)
	if who.Username != "bob" || !reflect.DeepEqual(who.Groups, []string{"team-a", "team-b"}) {
		t.Errorf("whoami = %+v, want bob with groups [team-a team-b]", who)
	}

	app.Do(nil, http.MethodGet, "/api/v1/whoami", nil).WantStatus(t, http.StatusUnauthorized)
}

func TestApp_AuthNoneUsesNoAuth(t *testing.T) {
	app := servertest.New(t, func(c *config.Config) { c.Auth.Type = "none" })

	app.Do(nil, http.MethodPost, "/api/v1/registry", map[string]any{"name": "repos"}).
		WantStatus(t, http.StatusCreated)
}

func TestApp_LogsAreCaptured(t *testing.T) {
	app := servertest.New(t)

	app.Do(nil, http.MethodGet, "/api/v1/registry/missing", nil).WantStatus(t, http.StatusNotFound)

	for _, line := range app.Logs() {
		if line["msg"] == "Request completed" &&
			line["endpoint"] == "/api/v1/registry/missing" &&
			line["status_code"] == float64(http.StatusNotFound) {
			return
		}
	}
	var msgs []string
	for _, line := range app.Logs() {
		msgs = append(msgs, line["msg"].(string))
	}
	t.Fatalf("no \"Request completed\" line with status 404; got %s", strings.Join(msgs, ", "))
}

// TestApp_DuplicateVersionConflict checks that publishing an existing
// version again is a 409 VERSION_ALREADY_EXISTS, as the API documents, not
// a 500.
func TestApp_DuplicateVersionConflict(t *testing.T) {
	app := servertest.New(t)
	alice := servertest.User("alice")
	version := map[string]any{
		"name":           "hotfix",
		"version":        "1.0.0",
		"checksum":       checksum,
		"url":            "https://example.com/hotfix-1.0.0.zip",
		"startPartition": 0,
		"endPartition":   9,
	}
	app.Do(alice, http.MethodPost, "/api/v1/registry", map[string]any{"name": "repos"}).
		WantStatus(t, http.StatusCreated)
	app.Do(alice, http.MethodPost, "/api/v1/registry/repos/package", map[string]any{"name": "hotfix"}).
		WantStatus(t, http.StatusCreated)
	app.Do(alice, http.MethodPost, "/api/v1/registry/repos/package/hotfix/version", version).
		WantStatus(t, http.StatusCreated)

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	app.Do(alice, http.MethodPost, "/api/v1/registry/repos/package/hotfix/version", version).
		WantStatus(t, http.StatusConflict).JSON(t, &body)
	if body.Error.Code != "VERSION_ALREADY_EXISTS" {
		t.Errorf("code = %q, want VERSION_ALREADY_EXISTS", body.Error.Code)
	}
}
