package handlers

import (
	"log/slog"
	"net/http"

	"github.com/criteo/command-launcher-registry/internal/auth"
	"github.com/criteo/command-launcher-registry/internal/models"
	"github.com/criteo/command-launcher-registry/internal/roles"
	"github.com/criteo/command-launcher-registry/internal/storage"
)

// Authorizer builds the storage guard of each write: the guard decides the
// write with roles.Decide on the data under the storage lock, and writes
// one audit line per decision.
type Authorizer struct {
	serverAdmins []string
	logger       *slog.Logger
}

// NewAuthorizer creates an authorizer. serverAdmins is the server-admin
// list in force: nil or empty when role enforcement is off, and then every
// write is allowed (and still audited). auditLogger receives the audit
// lines at Info; the server gives it one that ignores the configured log
// level (server.NewAuditLogger), so they are always visible.
func NewAuthorizer(serverAdmins []string, auditLogger *slog.Logger) *Authorizer {
	return &Authorizer{serverAdmins: serverAdmins, logger: auditLogger}
}

// Guard returns the storage guard for action on target by the caller that
// the auth middleware stored in r's context. A denied write fails with a
// *storage.ForbiddenError naming the missing role and its scope.
//
// newRoleList is the role list the write sets, nil when it leaves it alone:
// the registry admins for registry actions, the package maintainers for
// package actions. When set, the audit line also logs the old and new list.
func (a *Authorizer) Guard(r *http.Request, action roles.Action, target roles.Target, newRoleList *[]string) storage.Guard {
	caller := auth.UserFromContext(r.Context())
	return func(data *models.Storage) error {
		registry := data.Registries[target.Registry]
		d := roles.Decide(caller, a.serverAdmins, registry, action, target)
		a.audit(caller, action, target, d, registry, newRoleList)
		if !d.Allowed {
			return &storage.ForbiddenError{Role: string(d.Role), Target: d.Scope}
		}
		return nil
	}
}

// audit logs one decision. It never logs credentials: only the caller's
// username, never tokens or headers.
func (a *Authorizer) audit(caller *auth.User, action roles.Action, target roles.Target, d roles.Decision, registry *models.Registry, newRoleList *[]string) {
	username := ""
	if caller != nil {
		username = caller.Username
	}
	decision, roleKey := "allow", "matched_role"
	if !d.Allowed {
		decision, roleKey = "deny", "missing_role"
	}
	enforcement := "off"
	if d.Enforced {
		enforcement = "on"
	}

	attrs := []any{
		"caller", username,
		"action", string(action),
		"target", target.String(),
		"decision", decision,
		"enforcement", enforcement,
	}
	if d.Role != "" {
		attrs = append(attrs, roleKey, string(d.Role))
	}
	if newRoleList != nil {
		listName, old := oldRoleList(registry, action, target)
		attrs = append(attrs, "old_"+listName, nonNil(old), "new_"+listName, nonNil(*newRoleList))
	}
	a.logger.Info("Write audit", attrs...)
}

// oldRoleList returns the name and stored value of the role list an action
// on target sets: the registry admins for registry actions, the package
// maintainers for package actions. A missing registry or package has none.
func oldRoleList(registry *models.Registry, action roles.Action, target roles.Target) (string, []string) {
	switch action {
	case roles.CreateRegistry, roles.UpdateRegistry, roles.DeleteRegistry:
		if registry == nil {
			return "admins", nil
		}
		return "admins", registry.Admins
	default:
		if registry == nil || registry.Packages[target.Package] == nil {
			return "maintainers", nil
		}
		return "maintainers", registry.Packages[target.Package].Maintainers
	}
}

// nonNil makes an absent list log as [] rather than null.
func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
