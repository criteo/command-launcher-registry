package auth

import (
	"encoding/base64"
	"regexp"
)

// jwtPattern matches JWT tokens in the format: header.payload.signature
// Each part is base64url encoded (alphanumeric, dash, underscore)
var jwtPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)

// IsJWTToken checks if the given token matches JWT format
// JWT tokens have three base64url-encoded parts separated by dots
func IsJWTToken(token string) bool {
	if token == "" {
		return false
	}
	return jwtPattern.MatchString(token)
}

// EncodeToken prepares a token for HTTP transmission.
// JWT tokens are passed as-is, basic auth credentials are base64-encoded.
func EncodeToken(token string) string {
	if token == "" {
		return ""
	}
	if IsJWTToken(token) {
		return token
	}
	return base64.StdEncoding.EncodeToString([]byte(token))
}
