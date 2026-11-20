package cli

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/config"
)

// SetupFlags defines flags on the package-global ServerCmd, so it can run
// only once per process.
var setupFlagsOnce sync.Once

func setupServerFlags() { setupFlagsOnce.Do(SetupFlags) }

func TestServerFlags_Admins(t *testing.T) {
	setupServerFlags()

	flag := ServerCmd.Flags().Lookup("admins")
	if flag == nil {
		t.Fatal("server command has no --admins flag")
	}
	if !strings.Contains(flag.Usage, "COLA_REGISTRY_ADMINS") {
		t.Errorf("--admins help = %q, want it to name COLA_REGISTRY_ADMINS", flag.Usage)
	}
	if !strings.Contains(ServerCmd.Flags().FlagUsages(), "--admins") {
		t.Error("--help does not list --admins")
	}

	load := func() []string {
		t.Helper()
		cfg, err := config.LoadWithViper(v)
		if err != nil {
			t.Fatalf("LoadWithViper: %v", err)
		}
		return cfg.Admins
	}

	// The steps run in order: once the flag is set it stays set.
	t.Setenv("COLA_REGISTRY_ADMINS", "")
	if got := load(); got != nil {
		t.Errorf("default admins = %q, want none", got)
	}

	t.Setenv("COLA_REGISTRY_ADMINS", "alice, team-build ops")
	if got, want := load(), []string{"alice", "team-build", "ops"}; !reflect.DeepEqual(got, want) {
		t.Errorf("admins from env = %q, want %q", got, want)
	}

	if err := ServerCmd.Flags().Set("admins", "bob,ci"); err != nil {
		t.Fatalf("set --admins: %v", err)
	}
	if got, want := load(), []string{"bob", "ci"}; !reflect.DeepEqual(got, want) {
		t.Errorf("admins with flag and env = %q, want the flag's %q", got, want)
	}
}

func TestLogRoleSettings(t *testing.T) {
	tests := []struct {
		name         string
		admins       []string
		authType     string
		wantEnforced bool
		wantMsg      string
		wantWarning  bool
	}{
		{"on", []string{"alice", "team-build"}, "custom_jwt", true, "Role enforcement is on", false},
		{"off, no admins", nil, "custom_jwt", false, "Role enforcement is off", false},
		{"off, auth none", []string{"alice"}, "none", false, "Role enforcement is off", true},
		{"off, auth none, no admins", nil, "none", false, "Role enforcement is off", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			cfg := &config.Config{Admins: tt.admins, Auth: config.AuthConfig{Type: tt.authType}}

			logRoleSettings(cfg, logger)

			var info, warn map[string]any
			dec := json.NewDecoder(&buf)
			for dec.More() {
				var line map[string]any
				if err := dec.Decode(&line); err != nil {
					t.Fatalf("decode log line: %v", err)
				}
				switch line["level"] {
				case "INFO":
					info = line
				case "WARN":
					warn = line
				}
			}

			if info == nil {
				t.Fatalf("no info line logged: %s", buf.String())
			}
			if info["msg"] != tt.wantMsg || info["enforced"] != tt.wantEnforced {
				t.Errorf("info line = %v, want msg %q and enforced=%v", info, tt.wantMsg, tt.wantEnforced)
			}
			wantAdmins := []any{}
			for _, a := range tt.admins {
				wantAdmins = append(wantAdmins, a)
			}
			if !reflect.DeepEqual(info["server_admins"], wantAdmins) {
				t.Errorf("server_admins = %v, want %v", info["server_admins"], wantAdmins)
			}

			if tt.wantWarning != (warn != nil) {
				t.Fatalf("warning logged = %v, want %v; logs: %s", warn != nil, tt.wantWarning, buf.String())
			}
			if warn != nil && !strings.Contains(warn["msg"].(string), "roles are ignored") {
				t.Errorf("warning = %q, want it to say roles are ignored", warn["msg"])
			}
		})
	}
}

func TestServerFlags_RequireMaintainers(t *testing.T) {
	setupServerFlags()

	flag := ServerCmd.Flags().Lookup("require-maintainers")
	if flag == nil {
		t.Fatal("server command has no --require-maintainers flag")
	}
	if !strings.Contains(flag.Usage, "COLA_REGISTRY_REQUIRE_MAINTAINERS") {
		t.Errorf("--require-maintainers help = %q, want it to name COLA_REGISTRY_REQUIRE_MAINTAINERS", flag.Usage)
	}

	load := func() bool {
		t.Helper()
		cfg, err := config.LoadWithViper(v)
		if err != nil {
			t.Fatalf("LoadWithViper: %v", err)
		}
		return cfg.RequireMaintainers
	}

	// The steps run in order: once the flag is set it stays set.
	t.Setenv("COLA_REGISTRY_REQUIRE_MAINTAINERS", "")
	if load() {
		t.Error("default require_maintainers = true, want false")
	}

	t.Setenv("COLA_REGISTRY_REQUIRE_MAINTAINERS", "true")
	if !load() {
		t.Error("require_maintainers from env = false, want true")
	}

	if err := ServerCmd.Flags().Set("require-maintainers", "false"); err != nil {
		t.Fatalf("set --require-maintainers: %v", err)
	}
	if load() {
		t.Error("require_maintainers with --require-maintainers=false and env true = true, want the flag's false")
	}
}

func TestLogRequireMaintainers(t *testing.T) {
	for _, on := range []bool{true, false} {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))

		logRequireMaintainers(&config.Config{RequireMaintainers: on}, logger)

		var line map[string]any
		if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
			t.Fatalf("decode log line %q: %v", buf.String(), err)
		}
		wantMsg := "Required maintainers rule is off"
		if on {
			wantMsg = "Required maintainers rule is on"
		}
		if line["level"] != "INFO" || !strings.HasPrefix(line["msg"].(string), wantMsg) || line["require_maintainers"] != on {
			t.Errorf("on=%v: log line = %v, want INFO %q with require_maintainers=%v", on, line, wantMsg, on)
		}
	}
}
