package server_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/config"
	"github.com/criteo/command-launcher-registry/internal/server"
	"github.com/criteo/command-launcher-registry/internal/server/servertest"
)

// TestApp_AuditLinesIgnoreLogLevel checks that write audit lines are logged
// whatever logging.level is, while the level still filters ordinary lines.
func TestApp_AuditLinesIgnoreLogLevel(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error"} {
		t.Run(level, func(t *testing.T) {
			app := servertest.New(t, func(c *config.Config) {
				c.Admins = []string{"root"}
				c.Logging.Level = level
			})

			app.Do(serverAdminByName, http.MethodPost, "/api/v1/registry",
				map[string]any{"name": "repos"}).WantStatus(t, http.StatusCreated)
			app.Do(noRole, http.MethodPost, "/api/v1/registry/repos/package",
				map[string]any{"name": "hotfix"}).WantStatus(t, http.StatusForbidden)

			lines := auditLines(app)
			if len(lines) != 2 {
				t.Fatalf("got %d audit lines, want 2: %v", len(lines), lines)
			}
			wantAudit(t, lines[0], map[string]any{
				"level": "INFO", "caller": "root", "action": "create_registry", "target": "repos",
				"decision": "allow", "matched_role": "server admin", "enforcement": "on",
			})
			wantAudit(t, lines[1], map[string]any{
				"level": "INFO", "caller": "bob", "action": "create_package", "target": "repos/hotfix",
				"decision": "deny", "missing_role": "registry admin", "enforcement": "on",
			})

			// The configured level still applies to the server logger.
			infoLogged := false
			for _, line := range app.Logs() {
				if line["msg"] == "Registry created" {
					infoLogged = true
				}
			}
			if want := level == "debug" || level == "info"; infoLogged != want {
				t.Errorf("Info line \"Registry created\" logged = %v, want %v", infoLogged, want)
			}
		})
	}
}

// TestNewAuditLogger checks that the audit logger follows the log format
// and is enabled at Info, where the server logger at "error" is not.
func TestNewAuditLogger(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{"json", `"msg":"Write audit"`},
		{"text", `msg="Write audit"`},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			var buf bytes.Buffer
			server.NewAuditLogger(&buf, tt.format).Info("Write audit", "caller", "alice")
			server.NewLogger(&buf, "error", tt.format).Info("Dropped")

			out := buf.String()
			if !strings.Contains(out, tt.want) || !strings.Contains(out, "alice") {
				t.Errorf("audit output = %q, want it to contain %s and the caller", out, tt.want)
			}
			if strings.Contains(out, "Dropped") {
				t.Errorf("server logger at error logged an Info line: %q", out)
			}
			if !server.NewAuditLogger(&buf, tt.format).Enabled(context.Background(), slog.LevelInfo) {
				t.Error("audit logger not enabled at Info")
			}
		})
	}
}
