package server_test

import (
	"net/http"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/auth"
	"github.com/criteo/command-launcher-registry/internal/config"
	"github.com/criteo/command-launcher-registry/internal/server/servertest"
)

// TestApp_WhoamiServerAdmin checks the server_admin field of GET /whoami:
// true only when role enforcement is on and the caller's username or one of
// its groups is in the server-admin list.
func TestApp_WhoamiServerAdmin(t *testing.T) {
	admins := []string{"alice", "team-build"}
	tests := []struct {
		name     string
		admins   []string
		authType string
		caller   *auth.User
		want     bool
	}{
		{"matched by username", admins, "", servertest.User("alice"), true},
		{"matched by group", admins, "", servertest.User("bob", "ops", "team-build"), true},
		{"not in the list", admins, "", servertest.User("bob", "ops"), false},
		{"case-sensitive", admins, "", servertest.User("Alice", "Team-Build"), false},
		{"empty list", nil, "", servertest.User("alice", "team-build"), false},
		{"auth none ignores the list", []string{"anonymous"}, "none", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := servertest.New(t, func(c *config.Config) {
				c.Admins = tt.admins
				if tt.authType != "" {
					c.Auth.Type = tt.authType
				}
			})

			resp := app.Do(tt.caller, http.MethodGet, "/api/v1/whoami", nil).WantStatus(t, http.StatusOK)

			var body map[string]any
			resp.JSON(t, &body)
			got, present := body["server_admin"]
			if !present {
				t.Fatalf("whoami has no server_admin field: %s", resp.Body)
			}
			if got != tt.want {
				t.Errorf("server_admin = %v, want %v; body: %s", got, tt.want, resp.Body)
			}
		})
	}
}
