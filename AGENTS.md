# Development Guidelines for AI Agents

## Overview

This document provides principles, philosophy, and decision-making guidelines that cannot be easily extracted from reading the codebase. For implementation details, patterns, and structure, explore the code directly.

## Development Philosophy

### Professional Go Development

You are working with a professional Go developer who follows industry best practices:

- **Simplicity over cleverness**: Prefer straightforward, readable code over complex abstractions
- **Explicit is better than implicit**: Make dependencies and behavior clear
- **Error handling is part of the contract**: Always return and check errors, never ignore them
- **Standard library first**: Use Go standard library where possible before adding dependencies
- **Table-driven tests**: Use Go's testing patterns with table-driven test cases
- **Idiomatic Go**: Follow Go conventions (naming, structure, error handling)

### 12-Factor App Principles

This project strictly adheres to [12-factor app](https://12factor.net/) methodology:

1. **Configuration via environment**: All configuration through env vars, CLI flags optional
2. **No config files**: Never require configuration files
3. **Priority order**: CLI flags > Environment variables > Defaults
4. **Stateless processes**: Server is stateless, all state in storage backend
5. **Disposability**: Fast startup, graceful shutdown
6. **Logs to stdout**: Structured logging to stdout/stderr only
7. **Backing services**: Storage backends are attached resources (file, OCI, S3)
8. **Port binding**: Self-contained HTTP server, no external web server needed

#### Configuration Pattern

```go
// Configuration resolution always follows this pattern:
value := flagValue           // CLI flag (highest priority)
if value == "" {
    value = os.Getenv("ENV_VAR")  // Environment variable
}
if value == "" {
    value = defaultValue      // Default (lowest priority)
}
```

### Architecture Patterns

#### Server and CLI Separation

This project has TWO distinct binaries:
- **Server** (`cola-registry`): HTTP API server for hosting registries
- **CLI** (`cola-regctl`): Client tool for managing registries

These are completely separate applications with different concerns:
- Server focuses on API endpoints, authentication, storage
- CLI focuses on user experience, credential storage, output formatting

Never mix server and client code. Keep them in separate packages.

#### Storage Backend Abstraction

Storage backends implement a common interface but have fundamentally different characteristics:

- **File storage**: Simple, local, good for development and small deployments
- **OCI storage**: Distributed, versioned, good for GitOps workflows
- **S3 storage**: Scalable, durable, good for production deployments

When working with storage:
- All backends implement the same interface
- Handle backend-specific errors appropriately
- Each backend has its own connection/authentication mechanism
- Storage URI pattern: `scheme://host/path?query` with auto-detection

#### Authentication Extensibility

Authentication is pluggable with multiple strategies:
- **None**: No authentication (default, good for internal networks)
- **Basic**: Username/password with bcrypt hashes
- **LDAP**: Enterprise directory integration
- **Custom JWT**: External script validation for flexibility

When adding authentication features:
- Each auth type is independent
- Server startup validates auth configuration
- Failed auth returns 401, not 500
- Never log sensitive data (passwords, tokens)

## Decision-Making Guidelines

### When to Add Dependencies

Only add external dependencies when:
- Standard library cannot reasonably solve the problem
- The dependency is well-maintained and widely used in Go community
- The dependency simplifies code significantly

Every dependency in this project is intentional and justified. Review go.mod to see current dependencies.

### Error Handling Philosophy

- **Server errors (5xx)**: Only for bugs or infrastructure failures
- **Client errors (4xx)**: Validation failures, not found, unauthorized
- **Wrap context**: Use `fmt.Errorf("context: %w", err)` to add context
- **Don't panic**: Only panic in init functions or truly unrecoverable situations
- **Log before returning 500**: Always log internal errors before returning them

### API Design Principles

- **RESTful conventions**: Use proper HTTP methods and status codes
- **Consistent structure**: All responses follow same envelope pattern
- **Versioned endpoints**: All endpoints under `/api/v1/`
- **No breaking changes**: Once released, v1 API is stable
- **Cascade deletes**: Deleting a registry deletes packages and versions
- **Idempotent operations**: PUT/DELETE are idempotent where possible

### CLI User Experience

The CLI prioritizes user experience:

- **Sensible defaults**: Most flags optional, intelligent defaults
- **Multiple output formats**: Human-readable tables and machine-parseable JSON
- **Secure credentials**: OS-native credential storage (keychain, credential manager)
- **Confirmation prompts**: Dangerous operations (delete) require confirmation
- **Environment variables**: Support `COLA_REGISTRY_URL` and `COLA_REGISTRY_TOKEN`
- **Clear error messages**: Help users understand what went wrong and how to fix it

### Testing Strategy

- **Unit tests**: Test individual functions and methods
- **Integration tests**: Test storage backends with real services (OCI, S3, MinIO)
- **No mocks for external services**: Use real services in containers for integration tests
- **Table-driven tests**: Standard Go pattern for parametric tests
- **Test coverage**: Aim for high coverage but focus on critical paths

## Security Considerations

- **Secrets in environment**: Never hardcode secrets, always use env vars
- **Password hashing**: Use strong hashing algorithms with appropriate cost factors
- **Input validation**: Validate all inputs at boundaries (names, URLs, versions, etc.)
- **Path traversal**: Be careful with user-provided paths
- **Authentication bypass**: Ensure middleware is applied to all protected endpoints
- **Sensitive data logging**: Never log passwords, tokens, or other credentials

## Performance Considerations

- **In-memory state**: Storage backends load state into memory, optimize for small/medium datasets
- **Concurrency**: Handle concurrent access to shared state safely
- **Timeouts**: Set reasonable timeouts for all network operations
- **Storage efficiency**: Minimize writes to storage backends (they can be slow)
- **Graceful shutdown**: Handle shutdown signals properly to avoid data loss

## Development Workflow Expectations

### Making Changes

1. **Understand existing patterns**: Read similar code before adding new code
2. **Maintain consistency**: Match existing style and patterns
3. **Test your changes**: Add tests for new functionality
4. **Update documentation**: Update relevant docs (README, OpenAPI)
5. **Check formatting**: Run `make fmt` before committing

### Adding Features

- **Start with the API**: Define the API contract first (OpenAPI)
- **Implement storage**: Add storage operations if needed
- **Add handlers**: Implement HTTP handlers
- **Add CLI commands**: Add corresponding CLI commands
- **Write tests**: Unit and integration tests
- **Document**: Update README and spec

### Refactoring

- **Prefer small changes**: Make incremental improvements
- **Keep tests passing**: Ensure tests pass after each change
- **Don't over-engineer**: Only add abstraction when there's duplication
- **Backward compatibility**: Don't break existing API contracts

## What NOT to Do

- **Don't add config files**: Configuration is always via env vars/flags
- **Don't use global state**: Pass dependencies explicitly
- **Don't ignore errors**: Always check and handle errors
- **Don't add unnecessary dependencies**: Justify every new dependency
- **Don't mix concerns**: Keep server and client code separate
- **Don't hardcode values**: Use constants or configuration
- **Don't write brittle tests**: Tests should be maintainable
- **Don't skip validation**: Validate inputs at boundaries

## Communication with the Developer

### What to Clarify

Always ask when:
- Requirements are ambiguous
- Multiple valid approaches exist
- Breaking changes might be needed
- Security implications are unclear
- Performance tradeoffs need decisions

### What to Assume

You can assume:
- Developer understands Go and 12-factor principles
- Developer wants simple, maintainable code
- Developer values backward compatibility
- Developer prefers standard library over dependencies

## Reference Documentation

This document contains only principles and philosophy. For implementation details:

- **README.md**: User-facing documentation and quick start
- **docs/spec.md**: Complete specification
- **docs/openapi.yaml**: API contract
- **Makefile**: Build and development commands
- **go.mod**: Dependencies and versions
- **Codebase**: Package structure, patterns, and implementation details

When in doubt, prefer simplicity, follow Go conventions, and maintain consistency with existing code.
