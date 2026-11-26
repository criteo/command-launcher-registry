package config

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/criteo/command-launcher-registry/internal/branding"
)

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "empty token",
			token:    "",
			expected: "",
		},
		{
			name:     "non-empty token",
			token:    "my-secret-token",
			expected: "***",
		},
		{
			name:     "short token",
			token:    "x",
			expected: "***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Storage: StorageConfig{
					Token: tt.token,
				},
			}
			assert.Equal(t, tt.expected, cfg.MaskToken())
		})
	}
}

func TestValidate_StorageURI(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		wantError bool
		errMsg    string
	}{
		{
			name:      "valid file URI",
			uri:       "file://./data/registry.json",
			wantError: false,
		},
		{
			name:      "path without scheme (auto-prefixed)",
			uri:       "./data/registry.json",
			wantError: false,
		},
		{
			name:      "unsupported scheme",
			uri:       "ftp://host/path",
			wantError: true,
			errMsg:    "unsupported storage scheme",
		},
		{
			name:      "valid s3 scheme",
			uri:       "s3://s3.amazonaws.com/bucket/registry.json",
			wantError: false,
		},
		{
			name:      "valid oci scheme",
			uri:       "oci://registry.example.com/repo",
			wantError: false,
		},
		{
			name:      "empty URI",
			uri:       "",
			wantError: true,
			errMsg:    "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				Storage: StorageConfig{
					URI: tt.uri,
				},
				Auth: AuthConfig{
					Type: "none",
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
			}
			err := cfg.Validate()
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// loadAdmins builds the viper instance for the app named appName (which sets
// the env var prefix), binds an "--admins" flag as the server command does,
// sets the flag when flag is non-nil, and returns the loaded Admins.
func loadAdmins(t *testing.T, appName string, flag *string) []string {
	t.Helper()
	branding.Init(appName)
	t.Cleanup(func() { branding.Init("cola-registry") })

	v := NewViper()
	cmd := &cobra.Command{}
	cmd.Flags().String("admins", "", "")
	if err := v.BindPFlag("admins", cmd.Flags().Lookup("admins")); err != nil {
		t.Fatalf("bind flag: %v", err)
	}
	if flag != nil {
		if err := cmd.Flags().Set("admins", *flag); err != nil {
			t.Fatalf("set flag: %v", err)
		}
	}

	cfg, err := LoadWithViper(v)
	if err != nil {
		t.Fatalf("LoadWithViper: %v", err)
	}
	return cfg.Admins
}

func TestAdmins_Precedence(t *testing.T) {
	flag := func(s string) *string { return &s }
	tests := []struct {
		name    string
		appName string
		env     map[string]string
		flag    *string
		want    []string
	}{
		{name: "default is empty", appName: "cola-registry", want: nil},
		{
			name: "env", appName: "cola-registry",
			env:  map[string]string{"COLA_REGISTRY_ADMINS": "alice"},
			want: []string{"alice"},
		},
		{
			name: "flag beats env", appName: "cola-registry",
			env:  map[string]string{"COLA_REGISTRY_ADMINS": "alice"},
			flag: flag("bob"),
			want: []string{"bob"},
		},
		{
			name: "list parses commas and spaces", appName: "cola-registry",
			env:  map[string]string{"COLA_REGISTRY_ADMINS": "alice, team-build ops"},
			want: []string{"alice", "team-build", "ops"},
		},
		{
			name: "flag list parses commas and spaces", appName: "cola-registry",
			flag: flag("alice, team-build ops"),
			want: []string{"alice", "team-build", "ops"},
		},
		{
			name: "CDT prefix", appName: "cdt-registry",
			env:  map[string]string{"CDT_REGISTRY_ADMINS": "gu-cdt-maintainer svc-jenkins"},
			want: []string{"gu-cdt-maintainer", "svc-jenkins"},
		},
		{
			name: "CDT build ignores the COLA variable", appName: "cdt-registry",
			env:  map[string]string{"COLA_REGISTRY_ADMINS": "alice"},
			want: nil,
		},
		{
			name: "COLA build ignores the CDT variable", appName: "cola-registry",
			env:  map[string]string{"CDT_REGISTRY_ADMINS": "alice"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear both names so the caller's environment cannot leak in.
			t.Setenv("COLA_REGISTRY_ADMINS", "")
			t.Setenv("CDT_REGISTRY_ADMINS", "")
			for k, val := range tt.env {
				t.Setenv(k, val)
			}
			assert.Equal(t, tt.want, loadAdmins(t, tt.appName, tt.flag))
		})
	}
}

func TestRolesEnforced(t *testing.T) {
	tests := []struct {
		name     string
		admins   []string
		authType string
		want     bool
	}{
		{"admins and auth", []string{"alice"}, "custom_jwt", true},
		{"admins and basic auth", []string{"alice"}, "basic", true},
		{"no admins", nil, "custom_jwt", false},
		{"auth none", []string{"alice"}, "none", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Admins: tt.admins, Auth: AuthConfig{Type: tt.authType}}
			assert.Equal(t, tt.want, cfg.RolesEnforced())
		})
	}
}

// loadRequireMaintainers builds the viper instance for the app named appName,
// binds a "--require-maintainers" flag as the server command does, sets the
// flag when flag is non-nil, and returns the loaded RequireMaintainers.
func loadRequireMaintainers(t *testing.T, appName string, flag *string) (bool, error) {
	t.Helper()
	branding.Init(appName)
	t.Cleanup(func() { branding.Init("cola-registry") })

	v := NewViper()
	cmd := &cobra.Command{}
	cmd.Flags().Bool("require-maintainers", false, "")
	if err := v.BindPFlag("require_maintainers", cmd.Flags().Lookup("require-maintainers")); err != nil {
		t.Fatalf("bind flag: %v", err)
	}
	if flag != nil {
		if err := cmd.Flags().Set("require-maintainers", *flag); err != nil {
			t.Fatalf("set flag: %v", err)
		}
	}

	cfg, err := LoadWithViper(v)
	if err != nil {
		return false, err
	}
	return cfg.RequireMaintainers, nil
}

func TestRequireMaintainers_Precedence(t *testing.T) {
	flag := func(s string) *string { return &s }
	tests := []struct {
		name    string
		appName string
		env     map[string]string
		flag    *string
		want    bool
		wantErr bool
	}{
		{name: "default is off", appName: "cola-registry", want: false},
		{
			name: "env on", appName: "cola-registry",
			env:  map[string]string{"COLA_REGISTRY_REQUIRE_MAINTAINERS": "true"},
			want: true,
		},
		{
			name: "env off", appName: "cola-registry",
			env:  map[string]string{"COLA_REGISTRY_REQUIRE_MAINTAINERS": "false"},
			want: false,
		},
		{name: "flag on", appName: "cola-registry", flag: flag("true"), want: true},
		{
			name: "flag off beats env on", appName: "cola-registry",
			env:  map[string]string{"COLA_REGISTRY_REQUIRE_MAINTAINERS": "true"},
			flag: flag("false"),
			want: false,
		},
		{
			name: "flag on beats env off", appName: "cola-registry",
			env:  map[string]string{"COLA_REGISTRY_REQUIRE_MAINTAINERS": "false"},
			flag: flag("true"),
			want: true,
		},
		{
			name: "independent of admins", appName: "cola-registry",
			env:  map[string]string{"COLA_REGISTRY_ADMINS": "alice"},
			want: false,
		},
		{
			name: "CDT prefix", appName: "cdt-registry",
			env:  map[string]string{"CDT_REGISTRY_REQUIRE_MAINTAINERS": "true"},
			want: true,
		},
		{
			name: "CDT build ignores the COLA variable", appName: "cdt-registry",
			env:  map[string]string{"COLA_REGISTRY_REQUIRE_MAINTAINERS": "true"},
			want: false,
		},
		{
			name: "COLA build ignores the CDT variable", appName: "cola-registry",
			env:  map[string]string{"CDT_REGISTRY_REQUIRE_MAINTAINERS": "true"},
			want: false,
		},
		{
			name: "invalid env value is an error", appName: "cola-registry",
			env:     map[string]string{"COLA_REGISTRY_REQUIRE_MAINTAINERS": "maybe"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear every name so the caller's environment cannot leak in.
			for _, k := range []string{"COLA_REGISTRY_REQUIRE_MAINTAINERS", "CDT_REGISTRY_REQUIRE_MAINTAINERS", "COLA_REGISTRY_ADMINS", "CDT_REGISTRY_ADMINS"} {
				t.Setenv(k, "")
			}
			for k, val := range tt.env {
				t.Setenv(k, val)
			}
			got, err := loadRequireMaintainers(t, tt.appName, tt.flag)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
