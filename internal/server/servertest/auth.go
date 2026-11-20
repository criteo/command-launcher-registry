package servertest

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/criteo/command-launcher-registry/internal/auth"
)

// User returns a caller with the given username and groups, for App.Do.
func User(username string, groups ...string) *auth.User {
	return &auth.User{Username: username, Groups: groups}
}

// Token encodes user as a Bearer token that Authenticator accepts. App.Do
// sends it for you; use it directly only when calling the app over real
// HTTP.
func Token(user *auth.User) string {
	return url.Values{"user": {user.Username}, "group": user.Groups}.Encode()
}

// Authenticator is a test auth.Authenticator. It returns the user encoded
// in the request's Bearer token (see Token), and auth.ErrUnauthorized when
// the request has no token, a malformed one, or one without a username.
type Authenticator struct{}

// Authenticate implements auth.Authenticator.
func (Authenticator) Authenticate(r *http.Request) (*auth.User, error) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return nil, auth.ErrUnauthorized
	}
	values, err := url.ParseQuery(token)
	if err != nil || values.Get("user") == "" {
		return nil, auth.ErrUnauthorized
	}
	return &auth.User{Username: values.Get("user"), Groups: values["group"]}, nil
}

// Middleware implements auth.Authenticator with a pass-through middleware.
func (Authenticator) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}
