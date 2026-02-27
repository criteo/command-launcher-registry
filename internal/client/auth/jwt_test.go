package auth

import (
	"encoding/base64"
	"testing"
)

func TestIsJWTToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		// Valid JWT tokens
		{
			name:     "valid JWT token",
			token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			expected: true,
		},
		{
			name:     "JWT with dashes and underscores",
			token:    "abc-123_XYZ.def-456_UVW.ghi-789_RST",
			expected: true,
		},
		{
			name:     "minimal JWT",
			token:    "a.b.c",
			expected: true,
		},

		// Invalid JWT tokens
		{
			name:     "empty string",
			token:    "",
			expected: false,
		},
		{
			name:     "only two parts",
			token:    "header.payload",
			expected: false,
		},
		{
			name:     "four parts",
			token:    "part1.part2.part3.part4",
			expected: false,
		},
		{
			name:     "contains invalid characters",
			token:    "header@invalid.payload.signature",
			expected: false,
		},
		{
			name:     "contains spaces",
			token:    "header.pay load.signature",
			expected: false,
		},
		{
			name:     "basic auth credential (user:pass)",
			token:    "username:password",
			expected: false,
		},
		{
			name:     "base64 encoded basic auth",
			token:    "dXNlcm5hbWU6cGFzc3dvcmQ=",
			expected: false,
		},
		{
			name:     "random string",
			token:    "notajwttoken",
			expected: false,
		},
		{
			name:     "trailing dot",
			token:    "header.payload.signature.",
			expected: false,
		},
		{
			name:     "leading dot",
			token:    ".header.payload.signature",
			expected: false,
		},
		{
			name:     "double dots",
			token:    "header..payload.signature",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsJWTToken(tt.token)
			if result != tt.expected {
				t.Errorf("IsJWTToken(%q) = %v, expected %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestEncodeToken(t *testing.T) {
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
			name:     "JWT token passed as-is",
			token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc123",
			expected: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc123",
		},
		{
			name:     "basic auth gets base64 encoded",
			token:    "admin:password",
			expected: base64.StdEncoding.EncodeToString([]byte("admin:password")),
		},
		{
			name:     "simple username gets base64 encoded",
			token:    "username",
			expected: base64.StdEncoding.EncodeToString([]byte("username")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeToken(tt.token)
			if result != tt.expected {
				t.Errorf("EncodeToken(%q) = %q, expected %q", tt.token, result, tt.expected)
			}
		})
	}
}
