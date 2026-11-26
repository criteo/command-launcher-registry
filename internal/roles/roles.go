// Package roles decides who may write what, as ADR 0001 describes.
//
// There are three nested roles: server admin (the server-admin setting),
// registry admin (a registry's admins list) and package maintainer (a
// package's maintainers list). A higher role holds every right of the roles
// below it within its scope. Decide is a pure function of the caller, the
// server-admin list, the registry data, the action and the target; it does
// no I/O, so the storage layer can call it under its write lock.
package roles

import (
	"github.com/criteo/command-launcher-registry/internal/auth"
	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/criteo/command-launcher-registry/internal/principal"
)

// Role is one of the three roles. Its value is the human-readable name used
// in 403 messages and audit lines.
type Role string

// The roles, from the widest scope to the narrowest.
const (
	ServerAdmin       Role = "server admin"
	RegistryAdmin     Role = "registry admin"
	PackageMaintainer Role = "package maintainer"
)

// Action is a write on the registry server, one row of the ADR 0001 matrix.
type Action string

// The write actions.
const (
	CreateRegistry Action = "create_registry"
	UpdateRegistry Action = "update_registry"
	DeleteRegistry Action = "delete_registry"
	CreatePackage  Action = "create_package"
	UpdatePackage  Action = "update_package"
	DeletePackage  Action = "delete_package"
	CreateVersion  Action = "create_version"
	DeleteVersion  Action = "delete_version"
)

// RequiredRole returns the lowest role that may perform a: server admin to
// create or delete a registry; registry admin to update a registry or create
// or delete a package; package maintainer to update a package or publish or
// delete a version. An unknown action requires server admin.
func RequiredRole(a Action) Role {
	switch a {
	case UpdateRegistry, CreatePackage, DeletePackage:
		return RegistryAdmin
	case UpdatePackage, CreateVersion, DeleteVersion:
		return PackageMaintainer
	default:
		return ServerAdmin
	}
}

// Target names what an action writes. Package and Version are empty for
// registry actions, Version is empty for package actions.
type Target struct {
	Registry string
	Package  string
	Version  string
}

// String returns the target as a path: "repos", "repos/hotfix" or
// "repos/hotfix/1.0.0".
func (t Target) String() string {
	s := t.Registry
	if t.Package != "" {
		s += "/" + t.Package
	}
	if t.Version != "" {
		s += "/" + t.Version
	}
	return s
}

// Decision is the outcome of Decide.
type Decision struct {
	// Allowed reports whether the write may proceed.
	Allowed bool
	// Enforced reports whether roles were checked: false when the
	// server-admin list is empty, and then every write is allowed.
	Enforced bool
	// Role is the role the caller matched when allowed, or the role the
	// action requires when denied. It is empty when roles are not enforced.
	Role Role
	// Scope is what Role was checked on: the registry ("repos") for server
	// and registry admin, the package ("repos/hotfix") for package
	// maintainer. It is empty when roles are not enforced.
	Scope string
}

// Decide returns whether caller may perform action on target.
//
// serverAdmins is the server-admin list in force; when it is empty, roles
// are not enforced and every write is allowed. registry is the current data
// of target.Registry, nil when it does not exist (as for create registry).
// A nil caller holds no role. A principal matches the caller's username or
// any of its groups (principal.Matches). The narrowest matching role is
// reported.
func Decide(caller *auth.User, serverAdmins []string, registry *models.Registry, action Action, target Target) Decision {
	if len(serverAdmins) == 0 {
		return Decision{Allowed: true}
	}

	required := RequiredRole(action)
	var username string
	var groups []string
	if caller != nil {
		username, groups = caller.Username, caller.Groups
	}
	matches := func(list []string) bool {
		return caller != nil && principal.Matches(list, username, groups)
	}

	// Candidate roles from the narrowest that grants the action to the
	// widest, each with the principal list that holds it.
	type candidate struct {
		role  Role
		list  []string
		scope string
	}
	var candidates []candidate
	pkgScope := target.Registry + "/" + target.Package
	if required == PackageMaintainer && registry != nil {
		if pkg := registry.Packages[target.Package]; pkg != nil {
			candidates = append(candidates, candidate{PackageMaintainer, pkg.Maintainers, pkgScope})
		}
	}
	if required != ServerAdmin && registry != nil {
		candidates = append(candidates, candidate{RegistryAdmin, registry.Admins, target.Registry})
	}
	candidates = append(candidates, candidate{ServerAdmin, serverAdmins, target.Registry})

	for _, c := range candidates {
		if matches(c.list) {
			return Decision{Allowed: true, Enforced: true, Role: c.role, Scope: c.scope}
		}
	}

	scope := target.Registry
	if required == PackageMaintainer {
		scope = pkgScope
	}
	return Decision{Allowed: false, Enforced: true, Role: required, Scope: scope}
}
