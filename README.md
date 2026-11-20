# Command Launcher Registry

A complete solution for hosting and managing [Command Launcher](https://criteo.github.io/command-launcher/) remote registries, featuring both a REST API server and a powerful CLI client (`cola-regctl`) for seamless registry management.

## Features

### Server
- **Multi-Registry Support**: Isolated namespaces for different teams
- **Package & Version Management**: Full CRUD operations via REST API
- **Gradual Rollout**: Partition-based version distribution (0-9 range)
- **Command Launcher Compatible**: Serves index.json in CDT-compatible format
- **Authentication**: Optional Basic, LDAP or custom JWT authentication for write operations
- **Roles**: Opt-in write access by server admin, registry admin and package maintainer (users or groups); reads stay public
- **Pluggable Storage**: File-based JSON, OCI registry (ghcr.io, Docker Hub), or S3-compatible storage (AWS S3, MinIO, etc.)
- **Operational Ready**: Health checks, metrics, structured logging

### CLI Client (`cola-regctl`)
- **Full CRUD Operations**: Manage registries, packages, and versions from the command line
- **Secure Credential Storage**: OS-native keychains (macOS Keychain, Windows Credential Manager, Linux protected files)
- **Multiple Authentication Methods**: Username/password (Basic Auth) or JWT token (Bearer Auth)
- **Environment Variable Support**: `COLA_REGISTRY_URL` and `COLA_REGISTRY_TOKEN`
- **Multiple Output Formats**: Human-readable tables and machine-parseable JSON
- **Interactive Workflows**: Password prompts and deletion confirmations
- **Cross-Platform**: Works on macOS, Linux, and Windows

## Quick Start

### Prerequisites

- Go 1.24 or later
- Git

### Build

```bash
# Clone and build
git clone <repository-url>
cd command-launcher-registry
make build       # Build server
make build-cli   # Build CLI client
```

### Run Server

```bash
# Start server with default configuration
./bin/cola-registry server

# With explicit storage URI
./bin/cola-registry server --storage-uri file://./data/registry.json

# With path only (auto-prefixed with file://)
./bin/cola-registry server --storage-uri ./data/registry.json

# Full CLI configuration
./bin/cola-registry server \
  --storage-uri file://./data/registry.json \
  --port 8080 \
  --host 0.0.0.0 \
  --log-level info \
  --log-format json \
  --auth-type none

# Or with environment variables
export COLA_REGISTRY_STORAGE_URI=file://./data/registry.json
export COLA_REGISTRY_SERVER_PORT=8080
./bin/cola-registry server

# With OCI storage (GitHub Container Registry)
./bin/cola-registry server \
  --storage-uri oci://ghcr.io/yourusername/cola-registry-data \
  --storage-token ghp_your_github_token \
  --port 8080

# Or with environment variables for OCI
export COLA_REGISTRY_STORAGE_URI=oci://ghcr.io/yourusername/cola-registry-data
export COLA_REGISTRY_STORAGE_TOKEN=ghp_your_github_token
./bin/cola-registry server

# With S3 storage (AWS S3)
./bin/cola-registry server \
  --storage-uri s3://s3.us-east-1.amazonaws.com/mybucket/registry.json \
  --storage-token ACCESS_KEY:SECRET_KEY

# With S3 storage (MinIO for local development)
./bin/cola-registry server \
  --storage-uri s3+http://localhost:9000/mybucket/registry.json \
  --storage-token minioadmin:minioadmin
```

Server starts at `http://localhost:8080` by default.

### CLI Client Quick Start

```bash
# Set server URL and credentials via environment variables
export COLA_REGISTRY_URL=http://localhost:8080
export COLA_REGISTRY_TOKEN=admin:admin  # or use JWT token

# Create a registry
./bin/cola-regctl registry create my-tools --description "My tools registry"

# List registries
./bin/cola-regctl registry list

# Create a package
./bin/cola-regctl package create my-tools my-cli \
  --description "My CLI tool" \
  --maintainer "you, your-team"

# Publish a version
./bin/cola-regctl version create my-tools my-cli 1.0.0 \
  --checksum "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" \
  --url "https://downloads.example.com/my-cli-1.0.0.zip" \
  --start-partition 0 \
  --end-partition 9

# List versions
./bin/cola-regctl version list my-tools my-cli

# Get JSON output (for scripting)
./bin/cola-regctl registry list --json
```

### Populate Test Data

Using curl (direct API calls):
```bash
# Populate sample registry with packages and versions
./scripts/populate-test-data.sh

# Verify data
curl http://localhost:8080/api/v1/registry/company-tools/index.json | jq '.'

# Clean up test data
./scripts/clean-test-data.sh
```

Using CLI client:
```bash
# Populate sample registry with packages and versions
./scripts/populate-test-data-cli.sh

# Verify data using CLI
export COLA_REGISTRY_URL=http://localhost:8080
export COLA_REGISTRY_TOKEN=admin:admin
./bin/cola-regctl registry list
./bin/cola-regctl package list company-tools
./bin/cola-regctl version list company-tools deployment-cli

# Clean up test data
./scripts/clean-test-data-cli.sh
```

### Manual Testing

```bash
# Health check
curl http://localhost:8080/api/v1/health | jq '.'

# Metrics
curl http://localhost:8080/api/v1/metrics | jq '.'

# Create a registry (no auth required by default)
curl -X POST http://localhost:8080/api/v1/registry \
  -H "Content-Type: application/json" \
  -d '{"name":"build","description":"Build tools"}' | jq '.'

# Add a package
curl -X POST http://localhost:8080/api/v1/registry/build/package \
  -H "Content-Type: application/json" \
  -d '{"name":"hotfix","description":"Hotfix tool"}' | jq '.'

# Publish a version
curl -X POST http://localhost:8080/api/v1/registry/build/package/hotfix/version \
  -H "Content-Type: application/json" \
  -d '{
    "name":"hotfix",
    "version":"1.0.0",
    "checksum":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    "url":"https://example.com/hotfix-1.0.0.zip",
    "startPartition":0,
    "endPartition":9
  }' | jq '.'

# Fetch registry index (Command Launcher format)
curl http://localhost:8080/api/v1/registry/build/index.json | jq '.'
```

## Configuration

Configuration follows [12-factor app](https://12factor.net/) principles:
1. CLI flags (highest priority)
2. Environment variables
3. Defaults (lowest priority)

No configuration file is required.

### CLI Flags

```bash
cola-registry server [flags]

Flags:
  --storage-uri string     Storage URI (file:// for local, oci:// for OCI registry, s3:// for S3)
                           Default: file://./data/registry.json
  --storage-token string   Storage authentication token (required for OCI, optional for S3)
                           Default: (empty)
  --port int               Server port
                           Default: 8080
  --host string            Bind address
                           Default: 0.0.0.0
  --log-level string       Log level (debug|info|warn|error)
                           Default: info
  --log-format string      Log format (json|text)
                           Default: json
  --auth-type string       Authentication type (none|basic|ldap|custom_jwt)
                           Default: none
  --admins string          Server admins: users or groups, comma or space separated;
                           a non-empty list turns role enforcement on (see Roles and write access)
                           Default: (empty)
  --auth-ldap-server string      LDAP server URL (e.g., ldap://ldap.example.com)
  --auth-ldap-bind-dn string     LDAP bind DN for service account
  --auth-ldap-user-base-dn string  LDAP base DN for user searches
  --auth-ldap-timeout int          LDAP connection timeout in seconds
                                   Default: 30
  --require-maintainers    Reject packages with no maintainers (see Required Maintainers)
                           Default: false
```

### Environment Variables

All CLI flags have corresponding environment variables with `COLA_REGISTRY_` prefix:

```bash
export COLA_REGISTRY_STORAGE_URI=file://./data/registry.json
export COLA_REGISTRY_STORAGE_TOKEN=my-token        # Required for OCI storage
export COLA_REGISTRY_SERVER_PORT=8080
export COLA_REGISTRY_SERVER_HOST=0.0.0.0
export COLA_REGISTRY_LOGGING_LEVEL=info
export COLA_REGISTRY_LOGGING_FORMAT=json
export COLA_REGISTRY_AUTH_TYPE=basic
export COLA_REGISTRY_AUTH_USERS_FILE=./users.yaml  # Environment-only (no CLI flag)
export COLA_REGISTRY_REQUIRE_MAINTAINERS=false
export COLA_REGISTRY_ADMINS="alice, team-build"    # Server admins; turns role enforcement on
```

The prefix follows the binary's branding: a build with `PREFIX=cdt` (`cdt-registry`) reads `CDT_REGISTRY_*` instead, for example `CDT_REGISTRY_ADMINS`.

### Custom JWT Authentication

Custom JWT authentication delegates token validation to an external script. This allows integration with any JWT provider.

**Configuration:**
```bash
export COLA_REGISTRY_AUTH_TYPE=custom_jwt
export COLA_REGISTRY_AUTH_CUSTOM_JWT_SCRIPT=/path/to/jwt-validator.sh
export COLA_REGISTRY_AUTH_CUSTOM_JWT_REQUIRED_GROUP=my-admin-group  # Optional
```

**Custom JWT configuration options:**

| Setting | Environment Variable | CLI Flag | Default | Description |
|---------|---------------------|----------|---------|-------------|
| Script | `COLA_REGISTRY_AUTH_CUSTOM_JWT_SCRIPT` | `--auth-custom-jwt-script` | (required) | Path to JWT validator script |
| Required Group | `COLA_REGISTRY_AUTH_CUSTOM_JWT_REQUIRED_GROUP` | `--auth-custom-jwt-required-group` | - | Group required to access registry |

**Script requirements:**
- Must be executable and in PATH or specified as absolute path
- Receives JWT token as first argument
- Exit code 0 = valid token, non-zero = invalid token
- Output format: one group per line (optionally `username:value` on first line)

**Example script output:**
```
username:john.doe
admin-group
developers
readonly-users
```

Priority order: **CLI flags > Environment variables > Defaults**

### LDAP Authentication

LDAP authentication validates user credentials against an LDAP directory (e.g., Active Directory, OpenLDAP).

**Minimal configuration:**

```bash
export COLA_REGISTRY_AUTH_TYPE=ldap
export COLA_REGISTRY_AUTH_LDAP_SERVER=ldap://ldap.example.com
export COLA_REGISTRY_AUTH_LDAP_BIND_DN=cn=serviceaccount,dc=example,dc=com
export COLA_REGISTRY_AUTH_LDAP_BIND_PASSWORD=your-bind-password
export COLA_REGISTRY_AUTH_LDAP_USER_BASE_DN=ou=users,dc=example,dc=com

./bin/cola-registry server
```

**With group restriction:**

```bash
# Only allow members of the "registry-admins" group
export COLA_REGISTRY_AUTH_LDAP_REQUIRED_GROUP=cn=registry-admins,ou=groups,dc=example,dc=com
export COLA_REGISTRY_AUTH_LDAP_GROUP_BASE_DN=ou=groups,dc=example,dc=com
```

**LDAP configuration options:**

| Setting | Environment Variable | CLI Flag | Default | Description |
|---------|---------------------|----------|---------|-------------|
| Server | `COLA_REGISTRY_AUTH_LDAP_SERVER` | `--auth-ldap-server` | (required) | LDAP server URL |
| Bind DN | `COLA_REGISTRY_AUTH_LDAP_BIND_DN` | `--auth-ldap-bind-dn` | (required) | Service account DN |
| Bind Password | `COLA_REGISTRY_AUTH_LDAP_BIND_PASSWORD` | - | (required) | Service account password |
| User Base DN | `COLA_REGISTRY_AUTH_LDAP_USER_BASE_DN` | `--auth-ldap-user-base-dn` | (required) | Base DN for user searches |
| Timeout | `COLA_REGISTRY_AUTH_LDAP_TIMEOUT` | - | `30` | Connection timeout in seconds |
| User Filter | `COLA_REGISTRY_AUTH_LDAP_USER_FILTER` | - | `(uid=%s)` | LDAP filter for users |
| Group Base DN | `COLA_REGISTRY_AUTH_LDAP_GROUP_BASE_DN` | - | (empty) | Base DN for group searches |
| Group Filter | `COLA_REGISTRY_AUTH_LDAP_GROUP_FILTER` | - | `(member=%s)` | LDAP filter for groups |
| Required Group | `COLA_REGISTRY_AUTH_LDAP_REQUIRED_GROUP` | - | (empty) | DN of required group |

### Required Maintainers

An opt-in rule that keeps new packages from being created without an owner. It is off by default and independent of the server admins setting.

```bash
export COLA_REGISTRY_REQUIRE_MAINTAINERS=true   # or --require-maintainers
```

When it is on:

- Creating a package with `maintainers` missing or empty returns 400.
- A package PUT with `"maintainers": []` returns 400. A PUT that omits `maintainers` keeps the stored list and is allowed.
- The 400 message says that the server requires at least one maintainer.
- Existing packages with empty `maintainers` are left as they are. Under [ADR 0001](docs/adr/0001-role-based-write-access.md) only registry and server admins can change them, until their maintainers are backfilled.

The startup log says `Required maintainers rule is on` or `off`. In the CDT build the variable is `CDT_REGISTRY_REQUIRE_MAINTAINERS`.

Rollout: turn the rule on only once every automated client that creates packages passes maintainers (for CDT, after `release-v2` creates packages with `--maintainer`). Otherwise those clients can no longer create new packages.

### Storage URI

The storage backend is configured via URI:

```bash
# File storage
--storage-uri file://./data/registry.json       # Relative path
--storage-uri file:///var/data/registry.json    # Absolute path (Unix)
--storage-uri ./data/registry.json              # Auto-prefixed with file://

# OCI storage (GitHub Container Registry)
--storage-uri oci://ghcr.io/myorg/cola-registry-data
--storage-token ghp_xxxxxxxxxxxxxxxxxxxx

# OCI storage (Docker Hub)
--storage-uri oci://docker.io/myuser/cola-registry-data
--storage-token dckr_pat_xxxxxxxxx

# OCI storage (Azure Container Registry)
--storage-uri oci://myregistry.azurecr.io/cola-registry-data
--storage-token <acr-token>

# S3 storage (AWS S3)
--storage-uri s3://s3.us-east-1.amazonaws.com/mybucket/registry.json
--storage-token ACCESS_KEY:SECRET_KEY

# S3 storage with explicit region
--storage-uri s3://s3.amazonaws.com/mybucket/registry.json?region=us-east-1
--storage-token ACCESS_KEY:SECRET_KEY

# S3 storage (MinIO - local development)
--storage-uri s3+http://localhost:9000/mybucket/registry.json
--storage-token minioadmin:minioadmin

# S3 storage (DigitalOcean Spaces)
--storage-uri s3://nyc3.digitaloceanspaces.com/mybucket/registry.json
--storage-token ACCESS_KEY:SECRET_KEY

# S3 storage (Backblaze B2)
--storage-uri s3://s3.us-west-004.backblazeb2.com/mybucket/registry.json
--storage-token ACCESS_KEY:SECRET_KEY
```

**OCI Storage Notes**:
- OCI storage requires `--storage-token` or `COLA_REGISTRY_STORAGE_TOKEN` environment variable
- The registry data is stored as an OCI artifact with `latest` tag (overwritten on each write)
- Supports any OCI Distribution-compliant registry
- Token format is registry-specific (e.g., GitHub PAT for ghcr.io)

**S3 Storage Notes**:
- S3 storage uses `s3://` for HTTPS or `s3+http://` for HTTP connections
- Token format: `ACCESS_KEY:SECRET_KEY`
- Falls back to `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` environment variables if no token provided
- Supports IAM role authentication (leave token empty)
- Region is auto-detected from AWS endpoints or can be specified via `?region=` query parameter
- Compatible with any S3-compatible storage: AWS S3, MinIO, DigitalOcean Spaces, Backblaze B2, Wasabi, etc.

### Docker Usage

```bash
# Build the image
docker build -t cola-registry -f docker/Dockerfile .

# Run with file storage
docker run -p 8080:8080 \
  -e COLA_REGISTRY_STORAGE_URI=file:///data/registry.json \
  -v /host/data:/data \
  cola-registry

# Run with OCI storage
docker run -p 8080:8080 \
  -e COLA_REGISTRY_STORAGE_URI=oci://ghcr.io/myorg/cola-registry-data \
  -e COLA_REGISTRY_STORAGE_TOKEN=ghp_xxxxxxxxxxxx \
  cola-registry

# Run with S3 storage
docker run -p 8080:8080 \
  -e COLA_REGISTRY_STORAGE_URI=s3://s3.us-east-1.amazonaws.com/mybucket/registry.json \
  -e COLA_REGISTRY_STORAGE_TOKEN=ACCESS_KEY:SECRET_KEY \
  cola-registry
```

### Authentication Setup

To enable basic authentication:

```bash
# Generate password hash
./bin/cola-registry auth hash-password
# Enter password when prompted
# Copy the bcrypt hash

# Create users.yaml
cat > users.yaml <<EOF
users:
  - username: admin
    password: "$2a$10$..." # paste bcrypt hash here
EOF

export COLA_REGISTRY_AUTH_TYPE=basic
export COLA_REGISTRY_AUTH_USERS_FILE=./users.yaml

# Start server with authentication enabled
./bin/cola-registry server
```

Now write operations require authentication. To also require roles, set server admins (see [Roles and write access](#roles-and-write-access)); basic auth matches roles on the username:
```bash
# Create registry with auth
curl -u admin:yourpassword -X POST http://localhost:8080/api/v1/registry \
  -H "Content-Type: application/json" \
  -d '{"name":"secure","description":"Secure registry"}'
```

## Roles and write access

Reads (`GET`, `index.json`) are always public. Writes need an authenticated caller, and once role enforcement is on they also need a **role**. The decision and its full permission matrix are in [ADR 0001](./docs/adr/0001-role-based-write-access.md).

### Turning roles on

| Setting | Environment Variable | CLI Flag | Default |
|---------|---------------------|----------|---------|
| Server admins | `COLA_REGISTRY_ADMINS` | `--admins` | (empty) |

- The environment variable follows the binary's branding prefix: the CDT build (`cdt-registry`) reads `CDT_REGISTRY_ADMINS`. `cola-registry server --help` prints the name your build uses.
- Precedence is the usual one: `--admins` > `COLA_REGISTRY_ADMINS` > empty.
- **Enforcement is opt-in.** It is on only when the server-admin list is non-empty and `auth.type` is not `none`. With an empty list, any authenticated caller can write, as before. With `--auth-type none` and a non-empty list, the server logs a warning and ignores the roles.
- At startup the server logs `Role enforcement is on` (or `off`) with the parsed `server_admins` list. This line is logged at Info, so `--log-level warn` or `error` hides it.
- The `REQUIRED_GROUP` options of `custom_jwt` and `ldap` still work as a coarse gate in front of the roles.

```bash
# Server admins: one user and two groups
export COLA_REGISTRY_ADMINS="alice, team-build ops"
./bin/cola-registry server --auth-type custom_jwt --auth-custom-jwt-script /path/to/jwt-validator.sh

# Same with the flag, which wins over the environment variable
./bin/cola-registry server --admins "alice,team-build,ops" ...
```

### The three roles

| Role | Granted by | Can |
|------|-----------|-----|
| Server admin | `COLA_REGISTRY_ADMINS` / `--admins` only. No API call can grant it | Create and delete registries, plus everything a registry admin can do, on every registry |
| Registry admin | The registry's `admins` list | Update the registry (description, custom values, `admins`), create and delete its packages, plus everything a package maintainer can do, on every package of that registry |
| Package maintainer | The package's `maintainers` list | Update the package (description, custom values, `maintainers`), publish and delete its versions |

- A registry admin of one registry has no rights in another, and a maintainer of one package has no rights on another.
- Each role can edit the role list at its own level: registry admins add co-admins, maintainers add co-maintainers.
- Creating a package does not add the creator to `maintainers`. Set them with `--maintainer`.
- A package with an empty `maintainers` list can be changed only by its registry admins and the server admins. To stop new packages from being created without maintainers, see [Required Maintainers](#required-maintainers).

### Principals: users and groups

A **principal** is a username or a group name. It matches a caller whose username, or one of whose groups, is exactly equal to it (case-sensitive, no `user:`/`group:` prefix).

- Only `custom_jwt` reports groups (from the validator script's output). With `basic` and `ldap`, principals match the username only.
- Use usernames and groups, not emails. An email matches only if the authenticator reports the email as the username.
- Wherever principals are given as text (`--admins`, `COLA_REGISTRY_ADMINS`, and the CLI's `--admin` and `--maintainer`), the value is a list separated by commas, whitespace or both: `alice, team-build ops`. Empty entries are dropped, and the CLI flags are also repeatable.
- In the API, `admins` and `maintainers` are JSON arrays with one principal per element. The server rejects an element that is empty or contains a comma or whitespace with 400 `VALIDATION_ERROR`, whether or not enforcement is on:

```json
{"error":{"code":"VALIDATION_ERROR","message":"validation error on field 'admins': admins element \"a b\" must not contain whitespace (one user or group per element)"}}
```

### 401 or 403

| Status | When | Body | `cola-regctl` exit code |
|--------|------|------|-------------------------|
| 401 | A write with no or invalid credentials, or the caller fails `REQUIRED_GROUP` | `Unauthorized` (plain text) | 5 |
| 403 | An authenticated caller without the role the write needs | JSON error with code `FORBIDDEN` | 6 |

The 403 message names the missing role and its scope: the registry for server admin and registry admin, `registry/package` for package maintainer.

```json
{"error":{"code":"FORBIDDEN","message":"requires package maintainer on build/hotfix","details":{"required_role":"package maintainer","target":"build/hotfix"}}}
```

The checks run in this order: 401, then 400 (body validation), then 404 (registry or package missing), then 403, then 409 (already exists). The role check runs under the storage lock together with the write, so a role removed concurrently cannot let a write through.

### Audit log

Every authenticated write that reaches the role decision writes one `Write audit` line to stdout, allowed or denied, and also when enforcement is off:

```json
{"time":"...","level":"INFO","msg":"Write audit","caller":"bob","action":"update_package","target":"build/hotfix","decision":"allow","enforcement":"on","matched_role":"package maintainer","old_maintainers":["bob"],"new_maintainers":["bob","team-build"]}
{"time":"...","level":"INFO","msg":"Write audit","caller":"carol","action":"create_registry","target":"tools","decision":"deny","enforcement":"on","missing_role":"server admin"}
```

- `action` is one of `create_registry`, `update_registry`, `delete_registry`, `create_package`, `update_package`, `delete_package`, `create_version`, `delete_version`. `target` is the full path (`registry`, `registry/package` or `registry/package/version`).
- `matched_role` (allowed) or `missing_role` (denied) is absent when enforcement is off.
- A write that sets `admins` or `maintainers` also logs the old and the new list (`old_admins`/`new_admins`, `old_maintainers`/`new_maintainers`).
- The line holds the caller's username only: no groups, tokens, passwords or headers. Rejected credentials (401) and invalid bodies (400) produce no audit line; the request log covers them.
- **Audit lines ignore `logging.level`.** They use the same output and format (`--log-format`) as the other logs, but they are always written, even with `--log-level warn` or `error`.
- Nothing is written to storage.

### `whoami`

`GET /api/v1/whoami` returns `server_admin: true|false`. It is true only when enforcement is on and the caller matches the server-admin list. `cola-regctl whoami` prints it:

```
$ cola-regctl whoami
✓ Authenticated to http://localhost:8080 as alice
Server admin: yes
```

With `--json`, the output has a `server_admin` field.

### Updates keep omitted fields

`PUT` on a registry or a package changes only the fields present in the body (`description`, `custom_values`, and `admins` or `maintainers`):

- An absent field, or `null`, keeps the stored value, so a partial update never clears a role list by accident.
- An empty value (`[]`, `{}`, `""`) clears the field.
- `name` is optional. When present, it must equal the name in the URL (400 otherwise).

`cola-regctl registry update` and `package update` send only the flags you give. `--admin` and `--maintainer` replace the whole list, so pass the complete new list to add someone. `--clear-admins` and `--clear-maintainers` clear it:

```bash
# A maintainer adds a co-maintainer (the full list replaces the old one)
cola-regctl package update build hotfix --maintainer "bob, team-build, carol"

# Change only the description; maintainers and custom values are kept
cola-regctl package update build hotfix --description "Hotfix tool"
```

## CLI Client (`cola-regctl`)

The `cola-regctl` CLI provides a user-friendly interface for managing registries, packages, and versions.

### Installation

```bash
# Build the CLI
make build-cli

# Optionally install to system PATH
sudo cp bin/cola-regctl /usr/local/bin/
```

### Authentication

The CLI supports two authentication methods:
- **Username and password** (Basic Authentication)
- **JWT token** (Bearer Authentication)

Credential precedence (highest to lowest):
1. `--token` flag
2. `COLA_REGISTRY_TOKEN` environment variable
3. Stored credentials from `login` command

#### Interactive Login

```bash
# Login interactively (stores credentials securely)
./bin/cola-regctl login http://localhost:8080

# Choose authentication method:
# 1. Username and password
# 2. JWT token

# Option 1: Username/password
# Enter username: admin
# Enter password: ****

# Option 2: JWT token
# Enter JWT token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

# Check authentication status
./bin/cola-regctl whoami

# Logout (removes stored credentials)
./bin/cola-regctl logout
```

#### Environment Variables

```bash
# Set once, use everywhere
export COLA_REGISTRY_URL=http://localhost:8080

# Option 1: Basic authentication (username:password)
export COLA_REGISTRY_TOKEN=admin:admin

# Option 2: JWT token authentication
export COLA_REGISTRY_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0...

# Now all commands use these credentials automatically
./bin/cola-regctl registry list
./bin/cola-regctl package create my-registry my-package
```

#### Per-Command Authentication

```bash
# Override with flags (basic auth)
./bin/cola-regctl --url http://localhost:8080 --token admin:admin registry list

# Override with flags (JWT token)
./bin/cola-regctl --url http://localhost:8080 --token eyJhbGciOiJIUz... registry list
```

### Command Reference

#### Registry Management

```bash
# Create a registry
cola-regctl registry create <name> \
  --description "Description" \
  --admin "alice, team-build" \
  --custom-value "key=value"

# List all registries
cola-regctl registry list
cola-regctl registry list --json  # JSON output

# Get registry details
cola-regctl registry get <name>

# Update registry (only the flags given change; the rest is kept)
cola-regctl registry update <name> \
  --description "New description" \
  --admin "alice, team-build, bob"  # Replaces all admins
cola-regctl registry update <name> --clear-admins  # Clear all admins

# Delete registry (with confirmation)
cola-regctl registry delete <name>
cola-regctl registry delete <name> --yes  # Skip confirmation
```

#### Package Management

```bash
# Create a package
cola-regctl package create <registry> <package> \
  --description "Package description" \
  --maintainer "bob, team-build" \
  --custom-value "language=go" \
  --custom-value "repo=https://github.com/..."

# List packages
cola-regctl package list <registry>
cola-regctl package list <registry> --json

# Get package details
cola-regctl package get <registry> <package>

# Update package (only the flags given change; the rest is kept)
cola-regctl package update <registry> <package> \
  --description "New description" \
  --maintainer "bob, team-build, carol"  # Replaces all maintainers
cola-regctl package update <registry> <package> --clear-maintainers  # Clear all maintainers

# Delete package
cola-regctl package delete <registry> <package>
```

#### Version Management

```bash
# Publish a version
cola-regctl version create <registry> <package> <version> \
  --checksum "abc123..." \
  --url "https://downloads.example.com/package-1.0.0.zip" \
  --start-partition 0 \
  --end-partition 9

# List versions
cola-regctl version list <registry> <package>
cola-regctl version list <registry> <package> --json

# Get version details
cola-regctl version get <registry> <package> <version>

# Delete version
cola-regctl version delete <registry> <package> <version>
```

### Global Flags

All commands support these global flags:

- `--url <url>` - Server URL (or use `COLA_REGISTRY_URL` env var)
- `--token <token>` - Authentication token: `user:pass` for basic auth or JWT token for bearer auth (or use `COLA_REGISTRY_TOKEN` env var)
- `--json` - Output in JSON format (for scripting)
- `--verbose` - Enable verbose logging
- `--timeout <duration>` - HTTP request timeout (default: 30s)
- `--yes` / `-y` - Skip confirmation prompts

### Examples

#### Complete Workflow

```bash
# Set up environment
export COLA_REGISTRY_URL=http://localhost:8080
export COLA_REGISTRY_TOKEN=admin:admin

# Create a complete registry
cola-regctl registry create build-tools \
  --description "Build and deployment tools" \
  --admin "devops"

# Add a package
cola-regctl package create build-tools deployer \
  --description "Deployment automation tool" \
  --maintainer "devops" \
  --custom-value "language=go"

# Publish version 1.0.0 (for partitions 0-4 = 50% rollout)
cola-regctl version create build-tools deployer 1.0.0 \
  --checksum "abc123..." \
  --url "https://cdn.example.com/deployer-1.0.0.tar.gz" \
  --start-partition 0 \
  --end-partition 4

# Publish version 1.1.0 (for partitions 5-9 = 50% rollout)
cola-regctl version create build-tools deployer 1.1.0 \
  --checksum "def456..." \
  --url "https://cdn.example.com/deployer-1.1.0.tar.gz" \
  --start-partition 5 \
  --end-partition 9

# List all versions
cola-regctl version list build-tools deployer
```

#### Scripting with JSON Output

```bash
#!/bin/bash
# Get all registries and process with jq
registries=$(cola-regctl registry list --json | jq -r '.data[].name')

for registry in $registries; do
  echo "Packages in $registry:"
  cola-regctl package list "$registry" --json | jq -r '.data[].name'
done
```

### Credential Storage

The CLI stores credentials securely using OS-native mechanisms:

- **macOS**: Keychain (token) + file (URL)
- **Windows**: Credential Manager (token) + file (URL)
- **Linux**: Protected file (0600 permissions)

Credentials are stored at: `~/.config/cola-registry/credentials.yaml`

## AI Agent Skills

This repository includes [Agent Skills](https://agentskills.io/) that provide complete CLI references for AI coding agents (Claude Code, Cursor, OpenCode, and others).

### Install

```bash
npx skills add criteo/command-launcher-registry
```

This will prompt you to select which skills to install:

- **cola-registry-server** — Server CLI reference: all flags, env vars, storage URIs, API endpoints, auth types, roles, and validation rules
- **cola-regctl-client** — Client CLI reference: all commands, flags, env vars, credential storage, roles and exit codes, and validation rules

### What skills provide

Once installed, your AI agent gets full knowledge of:

- Every CLI flag and its corresponding environment variable
- All API endpoints with methods, auth requirements and the role each write needs
- Validation rules (name patterns, partition overlap, checksum format, etc.)
- Credential storage locations per platform
- Configuration precedence (flags > env vars > stored credentials)

This means you can ask your agent things like *"create a registry with two packages and roll out v2 to 30% of users"* and it will know the exact commands, flags, and constraints.

## Documentation

- **[Complete Specification](./docs/spec.md)** - Full specification covering both server and CLI client
- **[OpenAPI Contract](./docs/openapi.yaml)** - REST API specification
- **[ADR 0001: Role-based write access](./docs/adr/0001-role-based-write-access.md)** - The roles and their full permission matrix

### Endpoints

#### Operational
- `GET /api/v1/health` - Health check
- `GET /api/v1/metrics` - Server metrics
- `GET /api/v1/whoami` - The authenticated caller, its groups and `server_admin` (401 without valid credentials)

Writes need authentication (401 otherwise). With role enforcement on, each write also needs the role named below (403 otherwise); see [Roles and write access](#roles-and-write-access).

#### Registries
- `GET /api/v1/registry` - List all registries
- `POST /api/v1/registry` - Create registry (server admin)
- `GET /api/v1/registry/:name` - Get registry details
- `PUT /api/v1/registry/:name` - Update registry (registry admin; partial: fields absent from the body are kept)
- `DELETE /api/v1/registry/:name` - Delete registry (server admin, cascade)
- `GET /api/v1/registry/:name/index.json` - Get registry index (CDT format)

#### Packages
- `GET /api/v1/registry/:name/package` - List packages
- `POST /api/v1/registry/:name/package` - Create package (registry admin)
- `GET /api/v1/registry/:name/package/:package` - Get package details
- `PUT /api/v1/registry/:name/package/:package` - Update package (package maintainer; partial: fields absent from the body are kept)
- `DELETE /api/v1/registry/:name/package/:package` - Delete package (registry admin, cascade)

#### Versions
- `GET /api/v1/registry/:name/package/:package/version` - List versions
- `POST /api/v1/registry/:name/package/:package/version` - Create version (package maintainer)
- `GET /api/v1/registry/:name/package/:package/version/:version` - Get version details
- `DELETE /api/v1/registry/:name/package/:package/version/:version` - Delete version (package maintainer)

## Development

### Build Commands

```bash
make build        # Build server binary
make build-cli    # Build CLI client binary
make build-all    # Build both server and CLI
make clean        # Remove build artifacts
make test         # Run all tests
make test-cli     # Run CLI tests only
make run          # Build and run server
make fmt          # Format code
make fmt-check    # Check formatting (for CI)
make lint         # Run linter (go vet)
make install-cli  # Install CLI to $GOPATH/bin
make e2e-up       # Start the end-to-end stack (Docker)
make e2e          # Start it, run e2e scenarios and integration tests
make e2e-down     # Stop and remove the end-to-end stack
make help         # Show all targets
```

### End-to-end tests

`make e2e` checks what unit tests cannot: the real server image, S3 storage, authentication and the CLI working together. It needs Docker with Compose v2 (on macOS, for example, `colima start`).

What it starts (`tests/e2e/compose.yml`, project `clr-e2e`):

- the server image built from `docker/Dockerfile` with `PREFIX=cdt`, configured with a mix of `CDT_REGISTRY_*` env vars and flags;
- MinIO as its S3 storage, empty on every start;
- `custom_jwt` auth through `tests/e2e/auth/verify_jwt.sh`. This stub keeps the contract of the deployed validator: it checks issuer and expiry, takes the identity from `ctx:user:uid` or `ctx:service:name`, and reads groups from a fixture that stands in for IDM `member_of` (`tests/e2e/auth/idm-member-of.txt`). It does not check signatures. `REQUIRED_GROUP` is set, as in the deployment.

What it runs (`tests/e2e/run.sh`):

1. every scenario in `tests/e2e/scenarios/`, in name order, using `bin/cdt-regctl` and `curl`;
2. `go test ./tests/integration/...`, with the S3 tests pointed at the compose MinIO.

It prints one line per check and a summary. Logs go to `bin/e2e/` (`registry.log`, `integration.log`). The stack stays up afterwards, on `http://localhost:28080` (registry) and `localhost:29000` (MinIO); `make e2e-down` removes it. Every target is safe to rerun, and `make e2e` always starts from empty storage. Set `E2E_REGISTRY_PORT` or `E2E_MINIO_PORT` if those ports are taken.

To poke at the stack by hand:

```bash
source tests/e2e/lib.sh
export CDT_REGISTRY_URL=$E2E_URL CDT_REGISTRY_TOKEN=$(mint_token user e2e-writer)
bin/cdt-regctl registry list
```

To add a scenario, create `tests/e2e/scenarios/NN-name.sh`. `run.sh` sources it with the helpers of `tests/e2e/lib.sh` already loaded (`mint_token`, `run_regctl`, `check_regctl`, `check_status`, `check_contains`, …). The fixture identities and their groups are listed in `tests/e2e/auth/idm-member-of.txt`.

### Project Structure

```
cmd/
├── cola-registry/              # Server binary entry point
└── cola-regctl/                # CLI client binary entry point
internal/
├── server/                     # HTTP server
│   ├── handlers/               # HTTP handlers (health, index, registry, package, version)
│   └── middleware/             # Middleware (auth, cors, logging, ratelimit)
├── client/                     # CLI client
│   ├── commands/               # Cobra commands (registry, package, version, login, etc.)
│   ├── auth/                   # Credential storage (keyring, file)
│   ├── config/                 # URL resolution
│   ├── output/                 # Output formatters (table, JSON)
│   ├── prompts/                # Interactive prompts
│   ├── validation/             # Client-side validation
│   └── errors/                 # Exit codes
├── storage/                    # Storage backends (file, OCI, S3)
├── models/                     # Shared data models
├── auth/                       # Server authentication (basic, none)
├── cli/                        # Server CLI commands
├── config/                     # Server configuration
└── apierrors/                  # API error types
tests/
├── integration/                # Integration tests (OCI, S3, server)
└── e2e/                        # End-to-end stack and scenarios (make e2e)
scripts/                        # Test data scripts
docker/
└── Dockerfile                  # Multi-stage Docker build
docs/
├── openapi.yaml                # OpenAPI specification
└── quickstart.md               # Quick start guide
.github/
└── workflows/go.yml            # CI/CD pipeline
```
