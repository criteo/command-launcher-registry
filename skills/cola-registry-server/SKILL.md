---
name: cola-registry-server
description: Complete reference for the Command Launcher registry server CLI. Use when helping users start, configure, or troubleshoot the registry server, set up authentication, configure storage backends, or understand API endpoints. Covers all flags, environment variables, and auth types.
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

## Storage URIs

| Scheme | Example | Description |
|--------|---------|-------------|
| `file://` | `file://./data/registry.json` | Local JSON file storage (default). No token required. |
| `oci://` | `oci://ghcr.io/org/repo` | OCI registry (ghcr.io, docker.io, etc.). Requires `--storage-token`. |
| `s3://` | `s3://bucket-name/prefix` | S3-compatible storage (AWS S3, MinIO, DigitalOcean Spaces, Backblaze B2). Token optional (supports IAM). |
| `s3+http://` | `s3+http://localhost:9000/bucket/prefix` | S3 over plain HTTP (for local MinIO, etc.). |

## API Endpoints

Base path: `/api/v1`

### Public (no auth required)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Prometheus metrics |
| `GET` | `/whoami` | Current user info (auth optional) |
| `GET` | `/registry/{name}/index.json` | Download registry index |

### Registries (auth required)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/registry/` | List all registries |
| `POST` | `/registry/` | Create a registry |
| `GET` | `/registry/{name}` | Get registry details |
| `PUT` | `/registry/{name}` | Update a registry |
| `DELETE` | `/registry/{name}` | Delete a registry and all its contents |

### Packages (auth required for write operations)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/registry/{name}/package/` | List packages in a registry |
| `POST` | `/registry/{name}/package/` | Create a package |
| `GET` | `/registry/{name}/package/{pkg}` | Get package details |
| `PUT` | `/registry/{name}/package/{pkg}` | Update a package |
| `DELETE` | `/registry/{name}/package/{pkg}` | Delete a package and all versions |

### Versions (auth required for write operations)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/registry/{name}/package/{pkg}/version/` | List versions |
| `POST` | `/registry/{name}/package/{pkg}/version/` | Create a version |
| `GET` | `/registry/{name}/package/{pkg}/version/{ver}` | Get version details |
| `DELETE` | `/registry/{name}/package/{pkg}/version/{ver}` | Delete a version |

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

## Authentication Types

- **`none`** — No authentication. All API endpoints are open.
- **`basic`** — Username/password authentication. Passwords are bcrypt-hashed and stored in a `users.yaml` file. Use `auth hash-password` to generate hashes.
- **`ldap`** — LDAP server authentication. Validates credentials against an LDAP directory. Optionally enforces group membership.
- **`custom_jwt`** — JWT validation via an external script. The script receives the JWT and must exit 0 for valid tokens. Optionally enforces a group claim.

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

Generate a password hash for users.yaml:
```bash
cola-registry auth hash-password
```
