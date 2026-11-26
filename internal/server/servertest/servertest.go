// Package servertest runs the whole registry server in-process for tests.
//
// New builds the real application handler (server.NewHandler) over file
// storage in a temp dir, a test authenticator and loggers that write JSON
// lines to a buffer. Requests go straight to the handler, no network:
//
//	app := servertest.New(t)
//	app.Do(servertest.User("alice", "team-build"), http.MethodPost, "/api/v1/registry",
//		map[string]any{"name": "repos"}).WantStatus(t, http.StatusCreated)
//
// Each App has its own storage and its own rate limiter (100 requests per
// minute), so build one App per test case rather than sharing one across a
// large table.
package servertest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/auth"
	"github.com/criteo/command-launcher-registry/internal/config"
	"github.com/criteo/command-launcher-registry/internal/server"
	"github.com/criteo/command-launcher-registry/internal/storage"
)

// App is the registry server application running in-process.
type App struct {
	// Handler is the complete application handler. Wrap it in
	// httptest.NewServer to reach it over real HTTP (send Token(user) as a
	// Bearer token to authenticate).
	Handler http.Handler
	// Config is the configuration the app was built with.
	Config *config.Config
	// Store is the app's storage, a file in a temp dir.
	Store storage.Store

	t    testing.TB
	logs *syncBuffer
}

// New starts the app with file storage in t.TempDir(), the test
// authenticator and JSON loggers captured by Logs: the server logger at
// Config.Logging.Level ("debug" by default) and the audit logger, which
// ignores the level, as the server command builds them.
//
// The default config has auth type "custom_jwt", the only real
// authenticator that reports groups. Each configure function may change the
// config before the app is built. Setting Auth.Type to "none" swaps in the
// real no-auth authenticator, as the server command does.
func New(t testing.TB, configure ...func(*config.Config)) *App {
	t.Helper()

	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "127.0.0.1", Port: 8080},
		Storage: config.StorageConfig{URI: "file://" + filepath.Join(t.TempDir(), "registry.json")},
		Auth:    config.AuthConfig{Type: "custom_jwt"},
		Logging: config.LoggingConfig{Level: "debug", Format: "json"},
	}
	for _, f := range configure {
		f(cfg)
	}

	logs := &syncBuffer{}
	logger := server.NewLogger(logs, cfg.Logging.Level, "json")
	auditLogger := server.NewAuditLogger(logs, "json")

	uri, err := cfg.GetParsedStorageURI()
	if err != nil {
		t.Fatalf("servertest: parse storage URI: %v", err)
	}
	store, err := storage.NewStorage(uri, cfg.Storage.Token, logger)
	if err != nil {
		t.Fatalf("servertest: create storage: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("servertest: close storage: %v", err)
		}
	})

	var authenticator auth.Authenticator = Authenticator{}
	if cfg.Auth.Type == "none" {
		authenticator = auth.NewNoAuth()
	}

	return &App{
		Handler: server.NewHandler(cfg, logger, auditLogger, store, authenticator),
		Config:  cfg,
		Store:   store,
		t:       t,
		logs:    logs,
	}
}

// Response is a recorded HTTP response.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// Do sends one request to the app as user (nil means anonymous: no
// credentials). body is sent as is when it is a string or []byte, encoded as
// JSON otherwise, and omitted when nil.
func (a *App) Do(user *auth.User, method, path string, body any) *Response {
	a.t.Helper()

	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = strings.NewReader(b)
	case []byte:
		reader = bytes.NewReader(b)
	default:
		data, err := json.Marshal(b)
		if err != nil {
			a.t.Fatalf("servertest: encode request body: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if user != nil {
		req.Header.Set("Authorization", "Bearer "+Token(user))
	}

	rec := httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, req)
	return &Response{StatusCode: rec.Code, Header: rec.Header(), Body: rec.Body.Bytes()}
}

// Logs decodes every line the app has logged so far. Each line is one JSON
// object as written by slog (keys "msg", "level", and the attributes).
func (a *App) Logs() []map[string]any {
	a.t.Helper()

	var lines []map[string]any
	dec := json.NewDecoder(bytes.NewReader(a.logs.Bytes()))
	for dec.More() {
		var line map[string]any
		if err := dec.Decode(&line); err != nil {
			a.t.Fatalf("servertest: decode log line: %v", err)
		}
		lines = append(lines, line)
	}
	return lines
}

// WantStatus fails the test now unless the response has the given status.
// It returns r so a call can be chained.
func (r *Response) WantStatus(t testing.TB, status int) *Response {
	t.Helper()
	if r.StatusCode != status {
		t.Fatalf("status = %d, want %d; body: %s", r.StatusCode, status, r.Body)
	}
	return r
}

// JSON decodes the response body into v and fails the test if it cannot.
func (r *Response) JSON(t testing.TB, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode response body: %v; body: %s", err, r.Body)
	}
}

// syncBuffer is a bytes.Buffer safe for concurrent writes and reads.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return bytes.Clone(b.buf.Bytes())
}
