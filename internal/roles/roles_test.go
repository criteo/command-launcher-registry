package roles_test

import (
	"testing"

	"github.com/criteo/command-launcher-registry/internal/auth"
	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/criteo/command-launcher-registry/internal/roles"
)

// TestDecide_Matrix checks Decide against every cell of the ADR 0001
// permission matrix, with each role held by username and by group.
func TestDecide_Matrix(t *testing.T) {
	serverAdmins := []string{"root", "g-server-admins"}
	repos := &models.Registry{
		Name:   "repos",
		Admins: []string{"repo-admin", "g-repo-admins"},
		Packages: map[string]*models.Package{
			"hotfix": {Name: "hotfix", Maintainers: []string{"maint", "g-team-a"}},
			"other":  {Name: "other", Maintainers: []string{"other-maint"}},
		},
	}

	callers := []struct {
		name   string
		caller *auth.User
		holds  roles.Role // the role held on repos/hotfix, "" for none
	}{
		{"anonymous", nil, ""},
		{"no role", &auth.User{Username: "bob", Groups: []string{"g-misc"}}, ""},
		{"maintainer of another package", &auth.User{Username: "other-maint"}, ""},
		{"maintainer by username", &auth.User{Username: "maint"}, roles.PackageMaintainer},
		{"maintainer by group", &auth.User{Username: "carol", Groups: []string{"g-team-a"}}, roles.PackageMaintainer},
		{"registry admin by username", &auth.User{Username: "repo-admin"}, roles.RegistryAdmin},
		{"registry admin by group", &auth.User{Username: "dave", Groups: []string{"g-repo-admins"}}, roles.RegistryAdmin},
		{"server admin by username", &auth.User{Username: "root"}, roles.ServerAdmin},
		{"server admin by group", &auth.User{Username: "erin", Groups: []string{"g-server-admins"}}, roles.ServerAdmin},
		{"case differs", &auth.User{Username: "Root", Groups: []string{"G-Repo-Admins"}}, ""},
	}

	// rank orders the roles: a caller may act when it holds the required
	// role or a higher one.
	rank := map[roles.Role]int{"": 0, roles.PackageMaintainer: 1, roles.RegistryAdmin: 2, roles.ServerAdmin: 3}

	actions := []struct {
		action   roles.Action
		target   roles.Target
		registry *models.Registry
		required roles.Role
		scope    string
	}{
		{roles.CreateRegistry, roles.Target{Registry: "fresh"}, nil, roles.ServerAdmin, "fresh"},
		{roles.DeleteRegistry, roles.Target{Registry: "repos"}, repos, roles.ServerAdmin, "repos"},
		{roles.UpdateRegistry, roles.Target{Registry: "repos"}, repos, roles.RegistryAdmin, "repos"},
		{roles.CreatePackage, roles.Target{Registry: "repos", Package: "new"}, repos, roles.RegistryAdmin, "repos"},
		{roles.DeletePackage, roles.Target{Registry: "repos", Package: "hotfix"}, repos, roles.RegistryAdmin, "repos"},
		{roles.UpdatePackage, roles.Target{Registry: "repos", Package: "hotfix"}, repos, roles.PackageMaintainer, "repos/hotfix"},
		{roles.CreateVersion, roles.Target{Registry: "repos", Package: "hotfix", Version: "1.0.0"}, repos, roles.PackageMaintainer, "repos/hotfix"},
		{roles.DeleteVersion, roles.Target{Registry: "repos", Package: "hotfix", Version: "1.0.0"}, repos, roles.PackageMaintainer, "repos/hotfix"},
	}

	for _, a := range actions {
		for _, c := range callers {
			t.Run(string(a.action)+"/"+c.name, func(t *testing.T) {
				got := roles.Decide(c.caller, serverAdmins, a.registry, a.action, a.target)

				wantAllowed := rank[c.holds] >= rank[a.required]
				if got.Allowed != wantAllowed {
					t.Fatalf("Allowed = %v, want %v (decision %+v)", got.Allowed, wantAllowed, got)
				}
				if !got.Enforced {
					t.Errorf("Enforced = false, want true")
				}
				if wantAllowed {
					if got.Role != c.holds {
						t.Errorf("matched Role = %q, want %q", got.Role, c.holds)
					}
					return
				}
				if got.Role != a.required || got.Scope != a.scope {
					t.Errorf("missing role = %q on %q, want %q on %q", got.Role, got.Scope, a.required, a.scope)
				}
			})
		}
	}
}

// TestDecide_RegistryAdminScopedToItsRegistry checks that registry admin and
// package maintainer rights do not leak into another registry.
func TestDecide_RegistryAdminScopedToItsRegistry(t *testing.T) {
	other := &models.Registry{
		Name:     "other",
		Admins:   []string{"other-admin"},
		Packages: map[string]*models.Package{"tool": {Name: "tool", Maintainers: []string{"other-admin"}}},
	}
	repoAdmin := &auth.User{Username: "repo-admin"}

	for _, a := range []roles.Action{roles.UpdateRegistry, roles.CreatePackage, roles.UpdatePackage, roles.CreateVersion} {
		got := roles.Decide(repoAdmin, []string{"root"}, other, a, roles.Target{Registry: "other", Package: "tool"})
		if got.Allowed {
			t.Errorf("%s on other by repo-admin: allowed, want denied", a)
		}
	}
}

// TestDecide_NotEnforced checks that an empty server-admin list turns
// enforcement off: every caller, even anonymous, is allowed with no role.
func TestDecide_NotEnforced(t *testing.T) {
	for _, admins := range [][]string{nil, {}} {
		for _, caller := range []*auth.User{nil, {Username: "bob"}} {
			got := roles.Decide(caller, admins, nil, roles.CreateRegistry, roles.Target{Registry: "fresh"})
			want := roles.Decision{Allowed: true}
			if got != want {
				t.Errorf("Decide(%v, %v) = %+v, want %+v", caller, admins, got, want)
			}
		}
	}
}

// TestDecide_AnonymousNeverMatchesEmptyPrincipal checks that a caller
// without identity does not match a legacy empty principal in stored data.
func TestDecide_AnonymousNeverMatchesEmptyPrincipal(t *testing.T) {
	legacy := &models.Registry{Name: "legacy", Admins: []string{""}}
	got := roles.Decide(nil, []string{"root"}, legacy, roles.UpdateRegistry, roles.Target{Registry: "legacy"})
	if got.Allowed {
		t.Errorf("anonymous caller allowed by an empty principal: %+v", got)
	}
}

func TestTarget_String(t *testing.T) {
	tests := []struct {
		target roles.Target
		want   string
	}{
		{roles.Target{Registry: "repos"}, "repos"},
		{roles.Target{Registry: "repos", Package: "hotfix"}, "repos/hotfix"},
		{roles.Target{Registry: "repos", Package: "hotfix", Version: "1.0.0"}, "repos/hotfix/1.0.0"},
	}
	for _, tt := range tests {
		if got := tt.target.String(); got != tt.want {
			t.Errorf("%+v.String() = %q, want %q", tt.target, got, tt.want)
		}
	}
}
