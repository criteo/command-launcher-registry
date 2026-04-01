---
name: cola-regctl-client
description: Complete reference for the Command Launcher registry client CLI (regctl). Use when helping users manage registries, packages, and versions, authenticate with a server, or troubleshoot client connectivity. Covers all commands, flags, environment variables, and credential storage.
metadata:
  author: criteo
  version: "1.0.0"
---

# Command Launcher Registry Client (regctl)

Complete CLI reference for the registry client (`cola-regctl`).

> **Prefix note:** The binary and environment variables use the `cola` prefix by default. Alternative prefixes (e.g. `cdt`) can be configured at build time, changing the binary to `cdt-regctl` and env vars to `CDT_REGISTRY_*`. All examples below use the default `cola` prefix.

## Global Flags

Available on all commands:

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--url` | | string | _(empty)_ | Server URL (overrides stored/env) |
| `--token` | | string | _(empty)_ | Auth token: `user:password` for basic auth, or a JWT token (overrides stored/env) |
| `--json` | | bool | `false` | Output in JSON format |
| `--verbose` | | bool | `false` | Enable verbose HTTP request logging |
| `--timeout` | | duration | `30s` | HTTP request timeout |
| `--yes` | `-y` | bool | `false` | Skip confirmation prompts |

### Environment Variables

| Env Var | Description |
|---------|-------------|
| `COLA_REGISTRY_URL` | Server URL (used when `--url` flag is not set) |
| `COLA_REGISTRY_TOKEN` | Auth token (used when `--token` flag is not set) |

### Resolution Precedence

For both URL and token: **`--flag` > environment variable > stored credentials**

---

## Authentication Commands

### login — Authenticate with a server

```
cola-regctl login [server-url]
```

Authenticates with a registry server and stores credentials securely. If `server-url` is provided as an argument, it takes precedence over the environment variable.

Supports two authentication methods:
- **Username/password** — prompted interactively, stored as `user:password`
- **JWT token** — prompted interactively, detected by three dot-separated base64url parts

After login, tests the connection by calling `/api/v1/whoami`.

Only one server's credentials are stored at a time. Logging into a new server replaces existing credentials.

### logout — Remove stored credentials

```
cola-regctl logout
```

Removes stored URL and token. Idempotent — succeeds even if no credentials exist.

### whoami — Show authentication status

```
cola-regctl whoami
```

Displays the currently authenticated user and server information.

---

## Registry Commands

### registry create — Create a new registry

```
cola-regctl registry create <name> [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--description` | string | Registry description |
| `--admin` | string[] | Admin email address (repeatable) |
| `--custom-value` | string[] | Custom metadata as `key=value` (repeatable) |

### registry list — List all registries

```
cola-regctl registry list
```

Displays a table with columns: NAME, DESCRIPTION, PACKAGES.

### registry get — Get registry details

```
cola-regctl registry get <name>
```

### registry update — Update a registry

```
cola-regctl registry update <name> [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--description` | string | New description |
| `--admin` | string[] | Admin email (repeatable, replaces all existing admins) |
| `--custom-value` | string[] | Custom `key=value` (repeatable, replaces all existing values) |
| `--clear-admins` | bool | Remove all admins |
| `--clear-custom-values` | bool | Remove all custom values |

Cannot combine `--clear-admins` with `--admin`, or `--clear-custom-values` with `--custom-value`.

### registry delete — Delete a registry

```
cola-regctl registry delete <name>
```

Prompts for confirmation unless `--yes` is set. Deletes all packages and versions within the registry.

---

## Package Commands

### package create — Create a new package

```
cola-regctl package create <registry> <package> [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--description` | string | Package description |
| `--maintainer` | string[] | Maintainer email (repeatable) |
| `--custom-value` | string[] | Custom `key=value` (repeatable) |

### package list — List packages in a registry

```
cola-regctl package list <registry>
```

Displays a table with columns: NAME, DESCRIPTION, VERSIONS.

### package get — Get package details

```
cola-regctl package get <registry> <package>
```

### package update — Update a package

```
cola-regctl package update <registry> <package> [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--description` | string | New description |
| `--maintainer` | string[] | Maintainer email (repeatable, replaces all existing) |
| `--custom-value` | string[] | Custom `key=value` (repeatable, replaces all existing) |
| `--clear-maintainers` | bool | Remove all maintainers |
| `--clear-custom-values` | bool | Remove all custom values |

Cannot combine `--clear-maintainers` with `--maintainer`, or `--clear-custom-values` with `--custom-value`.

### package delete — Delete a package

```
cola-regctl package delete <registry> <package>
```

Prompts for confirmation unless `--yes` is set. Deletes all versions within the package.

---

## Version Commands

### version create — Create a new version

```
cola-regctl version create <registry> <package> <version> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--checksum` | string | _(required)_ | SHA256 checksum (exactly 64 hex characters) |
| `--url` | string | _(required)_ | Download URL for the version artifact |
| `--start-partition` | int | `0` | Start of partition range (0–9) |
| `--end-partition` | int | `9` | End of partition range (0–9) |

Partitions control gradual rollout: partition range `0-9` means all users, `0-0` means ~10% of users.

**Note:** The `--url` flag on this command is the **download URL**, not the server URL. It shadows the global `--url` flag, so the server URL must be provided via `COLA_REGISTRY_URL` environment variable or stored credentials when using this command.

### version list — List versions of a package

```
cola-regctl version list <registry> <package>
```

Displays a table with columns: VERSION, CHECKSUM, PARTITIONS.

### version get — Get version details

```
cola-regctl version get <registry> <package> <version>
```

### version delete — Delete a version

```
cola-regctl version delete <registry> <package> <version>
```

Prompts for confirmation unless `--yes` is set.

---

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

---

## Credential Storage

Credentials are stored per-platform in the `~/.config/cola-registry/` directory.

| Platform | Token Storage | URL Storage |
|----------|--------------|-------------|
| **macOS** | Keychain (service: `cola-registry`) | `~/.config/cola-registry/credentials.yaml` |
| **Windows** | Credential Manager (service: `cola-registry`) | `%APPDATA%\.config\cola-registry\credentials.yaml` |
| **Linux** | `~/.config/cola-registry/credentials.yaml` | Same file |

On Linux, the credentials file is created with `0600` permissions (owner read/write only).

Only one server's credentials are stored at a time.

---

## Examples

Login to a server:
```bash
cola-regctl login https://registry.example.com
```

List registries with environment variable:
```bash
export COLA_REGISTRY_URL=https://registry.example.com
export COLA_REGISTRY_TOKEN=admin:secret
cola-regctl registry list
```

Create a registry with metadata:
```bash
cola-regctl registry create my-tools \
  --description "Internal CLI tools" \
  --admin ops@example.com \
  --custom-value env=production
```

Publish a version with partitioned rollout (10% of users):
```bash
cola-regctl version create my-tools my-cli 2.1.0 \
  --checksum abc123...def456 \
  --url https://artifacts.example.com/my-cli-2.1.0.tar.gz \
  --start-partition 0 \
  --end-partition 0
```

Delete with auto-confirm:
```bash
cola-regctl version delete my-tools my-cli 1.0.0 --yes
```
