---
name: cola-registry-server
description: Complete reference for the Command Launcher registry server CLI. Use when helping users start, configure, or troubleshoot the registry server, set up authentication or role-based write access, configure storage backends, or understand API endpoints. Covers all flags, environment variables, auth types, and roles.
metadata:
  author: criteo
  version: "1.0.0"
---

# Command Launcher Registry Server

Complete CLI reference for the registry server (`cola-registry`).

> **Prefix note:** The binary and environment variables use the `cola` prefix by default. Alternative prefixes (e.g. `cdt`) can be configured at build time, changing the binary to `cdt-registry` and env vars to `CDT_REGISTRY_*`. All examples below use the default `cola` prefix.

## Commands

### server — Start the registry HTTP server

```
cola-registry server [flags]
```

Starts the HTTP server that serves Command Launcher registry indexes and provides a REST API for registry management.

### auth hash-password — Generate a bcrypt password hash

```
cola-registry auth hash-password
```

Prompts for a password (hidden input) and outputs a bcrypt hash for use in `users.yaml` when using basic authentication.

## Server Flags & Environment Variables

Configuration precedence: **CLI flags > Environment variables > Defaults**

### Storage

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--storage-uri` | `COLA_REGISTRY_STORAGE_URI` | `file://./data/registry.json` | Storage backend URI (see Storage URIs below) |
| `--storage-token` | `COLA_REGISTRY_STORAGE_TOKEN` | _(empty)_ | Authentication token for the storage backend |

### Server

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--port` | `COLA_REGISTRY_SERVER_PORT` | `8080` | HTTP listen port |
| `--host` | `COLA_REGISTRY_SERVER_HOST` | `0.0.0.0` | Bind address |

### Logging

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--log-level` | `COLA_REGISTRY_LOGGING_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `--log-format` | `COLA_REGISTRY_LOGGING_FORMAT` | `json` | Log format: `json`, `text` |

Audit lines (`Write audit`) ignore the log level: they are always written, in the same format, even with `--log-level warn` or `error`. See Roles and Write Access.

### Roles

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--admins` | `COLA_REGISTRY_ADMINS` | _(empty)_ | Server admins: users or groups, comma or space separated (e.g. `alice, team-build ops`). A non-empty list turns role enforcement on |

With the `cdt` prefix the variable is `CDT_REGISTRY_ADMINS`.

### Authentication

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--auth-type` | `COLA_REGISTRY_AUTH_TYPE` | `none` | Auth type: `none`, `basic`, `ldap`, `custom_jwt` |

#### Basic Auth Options

| Env Var (no CLI flag) | Default | Description |
|-----------------------|---------|-------------|
| `COLA_REGISTRY_AUTH_USERS_FILE` | `./users.yaml` | Path to users file with bcrypt-hashed passwords |

#### LDAP Auth Options

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--auth-ldap-server` | `COLA_REGISTRY_AUTH_LDAP_SERVER` | _(empty)_ | LDAP server URL (e.g. `ldap://ldap.example.com`) |
| `--auth-ldap-timeout` | `COLA_REGISTRY_AUTH_LDAP_TIMEOUT` | `30` | Connection timeout in seconds |
| `--auth-ldap-bind-dn` | `COLA_REGISTRY_AUTH_LDAP_BIND_DN` | _(empty)_ | Bind DN for service account |
| `--auth-ldap-user-base-dn` | `COLA_REGISTRY_AUTH_LDAP_USER_BASE_DN` | _(empty)_ | Base DN for user searches |

These additional LDAP settings are available only via environment variables:

| Env Var | Default | Description |
|---------|---------|-------------|
| `COLA_REGISTRY_AUTH_LDAP_BIND_PASSWORD` | _(empty)_ | Bind password for service account |
| `COLA_REGISTRY_AUTH_LDAP_USER_FILTER` | `(uid=%s)` | LDAP user search filter template |
| `COLA_REGISTRY_AUTH_LDAP_GROUP_BASE_DN` | _(empty)_ | Base DN for group searches |
| `COLA_REGISTRY_AUTH_LDAP_GROUP_FILTER` | `(member=%s)` | LDAP group membership filter template |
| `COLA_REGISTRY_AUTH_LDAP_REQUIRED_GROUP` | _(empty)_ | Required LDAP group for authorization |

#### Custom JWT Auth Options

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--auth-custom-jwt-script` | `COLA_REGISTRY_AUTH_CUSTOM_JWT_SCRIPT` | _(empty)_ | Path to external JWT validator script |
| `--auth-custom-jwt-required-group` | `COLA_REGISTRY_AUTH_CUSTOM_JWT_REQUIRED_GROUP` | _(empty)_ | Required group claim for authorization |

### Required Maintainers

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--require-maintainers` | `COLA_REGISTRY_REQUIRE_MAINTAINERS` | `false` | Reject a package with no maintainers (400) |

Opt-in and independent of the server admins setting. When on, creating a package with `maintainers` missing or empty returns 400, and so does a package PUT with `"maintainers": []`. A PUT that omits `maintainers` keeps the stored list and is allowed. The 400 message says the server requires at least one maintainer. Existing packages with empty `maintainers` are left as they are (admin-only under ADR 0001) until they are backfilled. The startup log line `Required maintainers rule is on|off` shows the setting. In the CDT build the variable is `CDT_REGISTRY_REQUIRE_MAINTAINERS`.

## Storage URIs

| Scheme | Example | Description |
|--------|---------|-------------|
| `file://` | `file://./data/registry.json` | Local JSON file storage (default). No token required. |
| `oci://` | `oci://ghcr.io/org/repo` | OCI registry (ghcr.io, docker.io, etc.). Requires `--storage-token`. |
| `s3://` | `s3://bucket-name/prefix` | S3-compatible storage (AWS S3, MinIO, DigitalOcean Spaces, Backblaze B2). Token optional (supports IAM). |
| `s3+http://` | `s3+http://localhost:9000/bucket/prefix` | S3 over plain HTTP (for local MinIO, etc.). |

## API Endpoints

Base path: `/api/v1`

Every `GET` is public. Every `POST`, `PUT` and `DELETE` needs authentication (401 otherwise) and, when role enforcement is on, the role in the Role column (403 otherwise).

### Operational

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Prometheus metrics |
| `GET` | `/whoami` | The caller's `username`, `groups` and `server_admin`. Needs valid credentials (401 otherwise) |
| `GET` | `/registry/{name}/index.json` | Download registry index |

### Registries

| Method | Path | Role | Description |
|--------|------|------|-------------|
| `GET` | `/registry/` | - | List all registries |
| `POST` | `/registry/` | server admin | Create a registry |
| `GET` | `/registry/{name}` | - | Get registry details |
| `PUT` | `/registry/{name}` | registry admin | Update a registry (partial: absent fields are kept) |
| `DELETE` | `/registry/{name}` | server admin | Delete a registry and all its contents |

### Packages

| Method | Path | Role | Description |
|--------|------|------|-------------|
| `GET` | `/registry/{name}/package/` | - | List packages in a registry |
| `POST` | `/registry/{name}/package/` | registry admin | Create a package |
| `GET` | `/registry/{name}/package/{pkg}` | - | Get package details |
| `PUT` | `/registry/{name}/package/{pkg}` | package maintainer | Update a package (partial: absent fields are kept) |
| `DELETE` | `/registry/{name}/package/{pkg}` | registry admin | Delete a package and all versions |

### Versions

| Method | Path | Role | Description |
|--------|------|------|-------------|
| `GET` | `/registry/{name}/package/{pkg}/version/` | - | List versions |
| `POST` | `/registry/{name}/package/{pkg}/version/` | package maintainer | Create a version |
| `GET` | `/registry/{name}/package/{pkg}/version/{ver}` | - | Get version details |
| `DELETE` | `/registry/{name}/package/{pkg}/version/{ver}` | package maintainer | Delete a version |

## Roles and Write Access

Full decision and permission matrix: `docs/adr/0001-role-based-write-access.md` (ADR 0001).

- **Opt-in.** Enforcement is on only when `--admins` / `COLA_REGISTRY_ADMINS` is non-empty and `--auth-type` is not `none`. Empty list: any authenticated caller can write. Auth `none` with a non-empty list: a startup warning, and roles are ignored.
- **Startup log:** `Role enforcement is on` (or `off`) with `server_admins` and `auth_type`, at Info level (hidden by `--log-level warn` or `error`).
- **Roles** (a higher role holds every right of the lower ones in its scope):
  - **Server admin**: listed in `--admins`. Never stored, so no API call grants it. Creates and deletes registries; registry admin of every registry.
  - **Registry admin**: in the registry's `admins`. Updates the registry (description, custom values, `admins`), creates and deletes its packages; maintainer of every package in it. No rights in other registries.
  - **Package maintainer**: in the package's `maintainers`. Updates the package (description, custom values, `maintainers`), publishes and deletes its versions.
  - The creator of a package is not added to `maintainers`. A package with empty `maintainers` is writable only by its registry admins and server admins.
- **Principal:** a username or group name, matched exactly (case-sensitive) against the caller's username or groups. Only `custom_jwt` reports groups; `basic` and `ldap` match the username only. Use usernames and groups, not emails.
- **REQUIRED_GROUP** (`custom_jwt`, `ldap`) still applies as a gate in front of the roles; failing it is 401.
- **Errors:** no or invalid credentials on a write → 401 `Unauthorized` (plain text). Authenticated without the role → 403 JSON, code `FORBIDDEN`, message `requires <role> on <scope>`, `details.required_role` and `details.target`. Example: `{"error":{"code":"FORBIDDEN","message":"requires registry admin on repos","details":{"required_role":"registry admin","target":"repos"}}}`. Order: 401 → 400 → 404 → 403 → 409.
- **Audit:** one `Write audit` line per authenticated write that reaches the role decision (allowed or denied, also with enforcement off), on stdout, with `caller`, `action` (`create_registry`, `update_registry`, `delete_registry`, `create_package`, `update_package`, `delete_package`, `create_version`, `delete_version`), `target`, `decision` (`allow`/`deny`), `enforcement` (`on`/`off`), `matched_role` or `missing_role` (absent when off), and `old_admins`/`new_admins` or `old_maintainers`/`new_maintainers` when the write sets a role list. No groups, tokens or headers. Audit lines ignore `logging.level`. Nothing is stored.
- **`whoami`:** `server_admin` is true only when enforcement is on and the caller matches `--admins`.
- **PUT is partial:** on a registry or a package, an absent (or `null`) field keeps the stored value; `[]`, `{}`, `""` clear it; `name` is optional and must match the URL when given.

## Validation Rules

### Names (registries and packages)

- Pattern: `^[a-z0-9][a-z0-9_-]*$` — lowercase alphanumeric, hyphens, underscores only. **No dots.**
- Length: 1–64 characters
- Must start with a letter or digit

### Versions

- Must be valid semver: `major[.minor[.patch]][-prerelease][+build]`
- Examples: `1`, `1.2`, `1.0.0`, `2.1.3-alpha`, `3.0.0-rc.1+build.42`

### Checksums

- Exactly 64 lowercase hex characters (SHA256)

### Partitions

- Must be >= 0 (no upper bound enforced)
- `startPartition` may be greater than `endPartition` (legacy convention for disabled/special rollout)
- Partition ranges **may overlap** across versions of the same package (multiple versions can cover the same partitions)

### Download URLs

- Must use `http://` or `https://` scheme
- Maximum 2048 characters

### Custom Values

- Maximum 20 key-value pairs per entity
- Key pattern: `^[a-zA-Z_][a-zA-Z0-9_-]{0,63}$`
- Value maximum: 1024 characters

### Descriptions

- Maximum 4096 characters

### Principals (`admins`, `maintainers`)

- JSON arrays, one user or group per element
- An element must be non-empty and contain no comma and no whitespace (400 `VALIDATION_ERROR` naming the field and element), whether or not enforcement is on
- `maintainers` is optional on package create (unless `--require-maintainers` is on)

### Maintainers (with `--require-maintainers`)

- Package create: `maintainers` must list at least one principal
- Package PUT: `"maintainers": []` is rejected; omitting `maintainers` keeps the stored list

## Authentication Types

- **`none`** — No authentication. All API endpoints are open.
- **`basic`** — Username/password authentication. Passwords are bcrypt-hashed and stored in a `users.yaml` file. Use `auth hash-password` to generate hashes. `users.yaml` has no roles; roles match the username.
- **`ldap`** — LDAP server authentication. Validates credentials against an LDAP directory. Optionally enforces group membership. Roles match the username only.
- **`custom_jwt`** — JWT validation via an external script. The script receives the JWT and must exit 0 for valid tokens. Optionally enforces a group claim. The script's group lines are the caller's groups, so roles can name groups.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Invalid configuration |
| 2 | Storage initialization failed |
| 3 | Server startup failed |

## Examples

Start with defaults (file storage, no auth, port 8080):
```bash
cola-registry server
```

Start with S3 storage and basic auth on a custom port:
```bash
cola-registry server \
  --storage-uri s3://my-bucket/registries \
  --auth-type basic \
  --port 9090
```

Same via environment variables:
```bash
export COLA_REGISTRY_STORAGE_URI=s3://my-bucket/registries
export COLA_REGISTRY_AUTH_TYPE=basic
export COLA_REGISTRY_SERVER_PORT=9090
cola-registry server
```

Turn on role enforcement with two server-admin principals (a user and a group):
```bash
cola-registry server \
  --auth-type custom_jwt \
  --auth-custom-jwt-script /path/to/jwt-validator.sh \
  --admins "alice, team-build"
```

Generate a password hash for users.yaml:
```bash
cola-registry auth hash-password
```
