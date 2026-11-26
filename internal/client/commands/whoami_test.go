package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// whoamiServer answers GET /api/v1/whoami with body.
func whoamiServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/whoami" {
			t.Errorf("request = %s %s, want GET /api/v1/whoami", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// captureStdout runs f and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan []byte)
	go func() {
		data, _ := io.ReadAll(r)
		done <- data
	}()
	defer func() { os.Stdout = orig }()
	f()
	w.Close()
	os.Stdout = orig
	return string(<-done)
}

func TestWhoami_ShowsServerAdmin(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		serverAdmin bool
		wantLine    string
	}{
		{"server admin", `{"username":"alice","groups":["team-build"],"server_admin":true}`, true, "Server admin: yes"},
		{"not server admin", `{"username":"bob","server_admin":false}`, false, "Server admin: no"},
		{"server without the field", `{"username":"bob"}`, false, "Server admin: no"},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/table", func(t *testing.T) {
			srv := whoamiServer(t, tt.body)
			out := captureStdout(t, func() {
				runCLI(t, "--url", srv.URL, "--token", "alice:secret", "whoami")
			})
			if !strings.Contains(out, tt.wantLine) {
				t.Errorf("output = %q, want a line %q", out, tt.wantLine)
			}
		})
		t.Run(tt.name+"/json", func(t *testing.T) {
			t.Cleanup(func() { flagJSON = false })
			srv := whoamiServer(t, tt.body)
			out := captureStdout(t, func() {
				runCLI(t, "--url", srv.URL, "--token", "alice:secret", "--json", "whoami")
			})
			var envelope struct {
				Data map[string]any `json:"data"`
			}
			if err := json.NewDecoder(bytes.NewReader([]byte(out))).Decode(&envelope); err != nil {
				t.Fatalf("decode JSON output %q: %v", out, err)
			}
			got := envelope.Data
			if got["server_admin"] != tt.serverAdmin || got["authenticated"] != true {
				t.Errorf("JSON output = %v, want authenticated=true and server_admin=%v", got, tt.serverAdmin)
			}
		})
	}
}
