package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/criteo/command-launcher-registry/internal/config"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// User represents an authenticated user
type User struct {
	Username string
}

// Authenticator defines the authentication interface
type Authenticator interface {
	// Authenticate validates request credentials and returns user info
	Authenticate(r *http.Request) (*User, error)

	// Middleware returns HTTP middleware for the auth method
	Middleware() func(http.Handler) http.Handler
}

// NewAuthenticator creates an authenticator based on config
func NewAuthenticator(cfg config.AuthConfig, logger *slog.Logger) (Authenticator, error) {
	switch cfg.Type {
	case "none":
		return NewNoAuth(), nil
	case "basic":
		return NewBasicAuth(cfg.UsersFile, logger)
	case "custom_jwt":
		return NewCustomJWTAuth(cfg.CustomJWT, logger)
	default:
		return nil, fmt.Errorf("unknown auth type: %s (valid options: none, basic, custom_jwt)", cfg.Type)
	}
}
