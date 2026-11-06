---
status: proposed
---

# Role-based write access: server admin, registry admin, package maintainer

Until now any authenticated caller could change any registry: the registry `admins` and package `maintainers` lists were stored but never checked. We enforce three nested roles instead, reusing those two lists rather than adding new fields, so the storage schema does not change. Reads stay public.

## Roles

- **Server admin**: listed in `COLA_REGISTRY_ADMINS` (or the matching flag). Never stored in the registry data, so no API call can grant it.
- **Registry admin**: a principal in a registry's `admins` list. Scoped to that registry only.
- **Package maintainer**: a principal in a package's `maintainers` list. Scoped to that package only.

Note on `maintainers`: the package `maintainers` field has been part of the registry since its first commit. It is a registry-only concept that Command Launcher's package and index schemas never had, so using it for the role adds no new field to the public API. A separate `custom_values` entry would only duplicate it.

A higher role holds every right of the roles below it, within its scope: a server admin is registry admin of every registry; a registry admin is maintainer of every package in it. A registry admin of one registry has no rights in another.

A **principal** is a plain string that matches when it equals the caller's username or one of the caller's groups. Groups come from the authenticator: today only `custom_jwt` reports them; `basic` and `ldap` match on username only. There is no `user:`/`group:` prefix; a username equal to a group name would match both, which we accept because the identity provider owns both namespaces.

Wherever principals are given as text, the value is a list separated by commas, whitespace, or both, for example `alice, team-build ops`. This covers `COLA_REGISTRY_ADMINS`, its flag, and the CLI `--admin` and `--maintainer` flags, which stay repeatable. Empty entries are dropped. The API and storage keep JSON arrays with one principal per element, and the server rejects an element that contains a comma or whitespace with 400. A principal therefore never contains a comma or whitespace.

## Permission matrix

| Action | Anonymous | Authenticated, no role | Package maintainer | Registry admin | Server admin |
|---|---|---|---|---|---|
| Read anything (GET, `index.json`) | yes | yes | yes | yes | yes |
| Create registry | no | no | no | no | yes |
| Delete registry | no | no | no | no | yes |
| Update registry description, custom values | no | no | no | yes | yes |
| Edit registry `admins` | no | no | no | yes | yes |
| Create package | no | no | no | yes | yes |
| Delete package | no | no | no | yes | yes |
| Update package description, custom values | no | no | yes | yes | yes |
| Edit package `maintainers` | no | no | yes | yes | yes |
| Publish version | no | no | yes | yes | yes |
| Delete version | no | no | yes | yes | yes |

Each role may edit the role list at its own level and below, so registry admins add co-admins and maintainers add co-maintainers without a server admin. At package creation the caller sets `maintainers`; the creator is not added implicitly.

## Enforcement and back-compat

- Enforcement is opt-in: role checks apply only when `COLA_REGISTRY_ADMINS` is non-empty. With it empty, any authenticated caller can write, as before; auth type `none` is unaffected.
- Unauthenticated write: 401, as before. Authenticated caller without the role: 403, naming the missing role and target (e.g. "requires registry admin on repos").
- The existing `REQUIRED_GROUP` option of `custom_jwt` and `ldap` stays as an optional coarse gate in front of the roles.
- The role check runs under the storage lock together with the mutation, so a role removed concurrently cannot let a write through.
- `GET /whoami` gains `server_admin: true|false`; no other API change.
- Every allowed or denied write is logged to stdout with caller, matched or missing role, action and target; role-list changes also log the old and new list. No audit data goes into storage.

## Considered options

- **A new `owner_groups` field on packages.** Rejected: it duplicates `maintainers`. Group-based ownership is just principals that are groups.
- **Maintainers in the package `custom_values` map (key `maintainers`).** Rejected: it would duplicate the existing `maintainers` field, and `custom_values` is free-form metadata for external tools that the server should not build access control on.
- **Maintainers in the package manifest metadata.** Rejected: whoever publishes a version writes its manifest, so maintainers would be self-asserted; a package has no manifest before its first version; and it would depend on the package manifest support in PR #20.
- **Server admins stored in the registry data.** Rejected: an API that can grant the top role needs its own bootstrap; configuration via environment avoids it.
- **Only lower levels may edit a role list.** Rejected: every co-maintainer change would need a registry admin. The audit log covers a maintainer removing others.
- **Explicit `group:` prefix on principals.** Rejected for simplicity; revisit if an authenticator returns groups whose names collide with usernames.

## Consequences

- Existing `admins`/`maintainers` entries become live. Entries written as emails will not match authenticators that report a uid, and must be rewritten.
- CLI help for `--admin` and `--maintainer` changes from "email" to "user or group".
- A package with an empty `maintainers` list can be changed only by its registry admins and server admins.
- Roles exist only in the registry server. `index.json` does not change, so Command Launcher is unaffected; packages used without a remote registry, or served from a static `index.json`, carry no maintainer information.
