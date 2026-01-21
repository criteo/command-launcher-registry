package auth

import (
	"errors"
	"net/http"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrInternal     = errors.New("internal error")
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
