package middleware

import (
	"net/http"

	"github.com/criteo/command-launcher-registry/internal/auth"
)

// RequireAuth returns middleware that requires authentication for write
// operations and stores the authenticated user in the request context (read
// it with auth.UserFromContext). Read operations (GET) are allowed without
// authentication.
func RequireAuth(authenticator auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if this is a write operation
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
				// Require authentication
				user, err := authenticator.Authenticate(r)
				if err != nil {
					w.Header().Set("WWW-Authenticate", `Basic realm="COLA Registry"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				r = r.WithContext(auth.WithUser(r.Context(), user))
			}

			next.ServeHTTP(w, r)
		})
	}
}
