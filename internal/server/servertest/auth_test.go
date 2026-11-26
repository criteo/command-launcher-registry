package servertest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/auth"
)

func TestAuthenticator(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   *auth.User
	}{
		{name: "user and groups", header: "Bearer " + Token(User("alice", "team-a", "team b")), want: User("alice", "team-a", "team b")},
		{name: "user without groups", header: "Bearer " + Token(User("bob")), want: &auth.User{Username: "bob"}},
		{name: "lowercase scheme", header: "bearer " + Token(User("bob")), want: &auth.User{Username: "bob"}},
		{name: "no header"},
		{name: "basic scheme", header: "Basic YWxpY2U6cHc="},
		{name: "empty username", header: "Bearer user=&group=team-a"},
		{name: "malformed token", header: "Bearer user=%zz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			got, err := Authenticator{}.Authenticate(req)

			if tt.want == nil {
				if !errors.Is(err, auth.ErrUnauthorized) {
					t.Fatalf("err = %v, want auth.ErrUnauthorized", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("user = %+v, want %+v", got, tt.want)
			}
		})
	}
}
