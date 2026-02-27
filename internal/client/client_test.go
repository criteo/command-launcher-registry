package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_AuthorizationHeader_JWT(t *testing.T) {
	// Create a test server that captures the Authorization header
	var capturedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Create client with JWT token
	jwtToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	client := NewClient(server.URL, jwtToken, 5*time.Second, false)

	// Make a request
	_, err := client.Get("/test")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Verify Bearer authentication was used
	expectedHeader := "Bearer " + jwtToken
	if capturedHeader != expectedHeader {
		t.Errorf("Authorization header = %q, expected %q", capturedHeader, expectedHeader)
	}
}

func TestClient_AuthorizationHeader_Basic(t *testing.T) {
	// Create a test server that captures the Authorization header
	var capturedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Create client with base64-encoded basic auth token
	basicToken := "dXNlcm5hbWU6cGFzc3dvcmQ=" // base64("username:password")
	client := NewClient(server.URL, basicToken, 5*time.Second, false)

	// Make a request
	_, err := client.Get("/test")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Verify Basic authentication was used
	expectedHeader := "Basic " + basicToken
	if capturedHeader != expectedHeader {
		t.Errorf("Authorization header = %q, expected %q", capturedHeader, expectedHeader)
	}
}

func TestClient_AuthorizationHeader_Empty(t *testing.T) {
	// Create a test server that captures the Authorization header
	var capturedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Create client with no token
	client := NewClient(server.URL, "", 5*time.Second, false)

	// Make a request
	_, err := client.Get("/test")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Verify no Authorization header was set
	if capturedHeader != "" {
		t.Errorf("Authorization header = %q, expected empty string", capturedHeader)
	}
}

func TestClient_AllMethodsUseAuth(t *testing.T) {
	jwtToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"

	tests := []struct {
		name   string
		method func(*Client) (*http.Response, error)
	}{
		{
			name: "GET",
			method: func(c *Client) (*http.Response, error) {
				return c.Get("/test")
			},
		},
		{
			name: "POST",
			method: func(c *Client) (*http.Response, error) {
				return c.Post("/test", map[string]string{"key": "value"})
			},
		},
		{
			name: "PUT",
			method: func(c *Client) (*http.Response, error) {
				return c.Put("/test", map[string]string{"key": "value"})
			},
		},
		{
			name: "DELETE",
			method: func(c *Client) (*http.Response, error) {
				return c.Delete("/test")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedHeader string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedHeader = r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "ok"}`))
			}))
			defer server.Close()

			client := NewClient(server.URL, jwtToken, 5*time.Second, false)
			_, err := tt.method(client)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			expectedHeader := "Bearer " + jwtToken
			if capturedHeader != expectedHeader {
				t.Errorf("%s: Authorization header = %q, expected %q", tt.name, capturedHeader, expectedHeader)
			}
		})
	}
}
