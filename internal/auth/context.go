package auth

import "context"

type userKey struct{}

// WithUser returns a copy of ctx that carries the authenticated user.
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}

// UserFromContext returns the authenticated user stored by WithUser, or nil
// when the request was not authenticated.
func UserFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(userKey{}).(*User)
	return user
}
