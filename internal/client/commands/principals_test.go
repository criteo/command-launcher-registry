package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// recordedRequest is one request the fake server received.
type recordedRequest struct {
	Method string
	Path   string
	Body   map[string]any
}

// fakeServer answers every request with success and records it.
func fakeServer(t *testing.T) (*httptest.Server, *[]recordedRequest) {
	t.Helper()
	var got []recordedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		rec := recordedRequest{Method: r.Method, Path: r.URL.Path}
		if len(data) > 0 {
			if err := json.Unmarshal(data, &rec.Body); err != nil {
				t.Errorf("decode request body %q: %v", data, err)
			}
		}
		got = append(got, rec)

		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

// runCLI runs the root command with args. The persistent flags are defined
// by Execute, so the first run goes through it.
func runCLI(t *testing.T, args ...string) {
	t.Helper()
	// Command flags are package globals: reset the ones these tests set.
	pkgMaintainers, regAdmins = nil, nil
	pkgDescription, regDescription = "", ""
	pkgClearMaint, regClearAdmins = false, false

	rootCmd.SetArgs(args)
	var err error
	if rootCmd.PersistentFlags().Lookup("url") == nil {
		err = Execute()
	} else {
		err = rootCmd.Execute()
	}
	if err != nil {
		t.Fatalf("%v: %v", args, err)
	}
}

func TestPrincipalFlags_SendParsedList(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		method string
		path   string
		field  string
		want   []string
	}{
		{
			name:   "package create",
			args:   []string{"package", "create", "repos", "hotfix", "--maintainer", "alice, team-build", "--maintainer", "ops"},
			method: http.MethodPost, path: "/api/v1/registry/repos/package",
			field: "maintainers", want: []string{"alice", "team-build", "ops"},
		},
		{
			name:   "package update",
			args:   []string{"package", "update", "repos", "hotfix", "--maintainer", "alice, team-build", "--maintainer", "ops"},
			method: http.MethodPut, path: "/api/v1/registry/repos/package/hotfix",
			field: "maintainers", want: []string{"alice", "team-build", "ops"},
		},
		{
			name:   "registry create",
			args:   []string{"registry", "create", "repos", "--admin", " alice,,team-build\tops "},
			method: http.MethodPost, path: "/api/v1/registry",
			field: "admins", want: []string{"alice", "team-build", "ops"},
		},
		{
			name:   "registry update",
			args:   []string{"registry", "update", "repos", "--admin", "alice", "--admin", "team-build ops"},
			method: http.MethodPut, path: "/api/v1/registry/repos",
			field: "admins", want: []string{"alice", "team-build", "ops"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, got := fakeServer(t)
			runCLI(t, append([]string{"--url", srv.URL, "--token", "alice:secret", "--yes"}, tt.args...)...)

			if len(*got) != 1 {
				t.Fatalf("server got %d requests, want 1: %+v", len(*got), *got)
			}
			req := (*got)[0]
			if req.Method != tt.method || req.Path != tt.path {
				t.Errorf("request = %s %s, want %s %s", req.Method, req.Path, tt.method, tt.path)
			}
			raw, _ := req.Body[tt.field].([]any)
			var list []string
			for _, v := range raw {
				s, _ := v.(string)
				list = append(list, s)
			}
			if !slices.Equal(list, tt.want) {
				t.Errorf("%s = %#v, want %q", tt.field, req.Body[tt.field], tt.want)
			}
		})
	}
}

func TestPrincipalFlags_HelpSaysUserOrGroup(t *testing.T) {
	flags := []struct {
		cmd  *cobra.Command
		flag string
	}{
		{packageCreateCmd, "maintainer"},
		{packageUpdateCmd, "maintainer"},
		{registryCreateCmd, "admin"},
		{registryUpdateCmd, "admin"},
	}
	for _, f := range flags {
		usage := f.cmd.Flags().Lookup(f.flag).Usage
		if !strings.Contains(usage, "user or group (repeatable; comma or space separated") {
			t.Errorf("%s --%s help = %q, want it to say user or group, repeatable, comma or space separated",
				f.cmd.CommandPath(), f.flag, usage)
		}
		if strings.Contains(strings.ToLower(usage), "email") {
			t.Errorf("%s --%s help = %q, still says email", f.cmd.CommandPath(), f.flag, usage)
		}
	}
}
