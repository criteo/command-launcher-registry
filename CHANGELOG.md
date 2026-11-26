# Changelog

All notable user-visible changes to this project are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions are the git tags (`X.Y.Z-alpha` so far).

The top section is the next release. While it is not tagged, its heading says `Unreleased` instead of a date; the date is set to the tag date when the release is tagged. Full release notes for a version, with upgrade steps for operators, live in [`docs/release-notes/`](docs/release-notes/), which also says how to publish them as the GitHub release body.

Entries up to `0.5.0-alpha` are taken from the GitHub releases, which listed the merged pull requests.

## [0.6.0-alpha] - Unreleased

Role-based write access (ADR 0001). Read the [release notes](docs/release-notes/0.6.0-alpha.md) before upgrading: PUT, principal validation, audit logging and the duplicate-version status change for every deployment, even with roles off.

### Added

- Server-admin setting `--admins` / `COLA_REGISTRY_ADMINS` (`CDT_REGISTRY_ADMINS` in the CDT build). A non-empty list turns role enforcement on when an authenticator is configured.
- Role enforcement on registry, package and version writes: server admin, registry admin, package maintainer. A write without the role gets 403 `FORBIDDEN`.
- `GET /api/v1/whoami` returns `server_admin`; `cola-regctl whoami` prints it.
- A `Write audit` line on stdout for every authenticated write, whatever the log level.
- Opt-in `--require-maintainers` / `COLA_REGISTRY_REQUIRE_MAINTAINERS`: a package must have at least one maintainer.
- CLI `--admin` / `--maintainer` accept comma- and space-separated lists.
- `make e2e`: end-to-end loop with the CDT image, MinIO S3 storage and `custom_jwt` auth.

### Changed

- Registry and package PUT keep omitted or `null` fields; an empty value (`[]`, `{}`, `""`) clears. `name` is optional in PUT bodies.
- `admins` / `maintainers` elements that are empty or contain a comma or whitespace are rejected with 400.

### Fixed

- `cola-regctl registry update` and `cola-regctl package update` work again (they always failed with 400).
- Publishing a version that already exists returns 409 `VERSION_ALREADY_EXISTS` instead of 500 `STORAGE_UNAVAILABLE`.

### Docs

- README "Roles and write access", quickstart, agent skills, presentations and test-data scripts use usernames and groups instead of emails.
- Agent Skills documentation ([#17](https://github.com/criteo/command-launcher-registry/pull/17)).
- ADR 0001 (role-based write access) and the glossary ([#22](https://github.com/criteo/command-launcher-registry/pull/22)). The ADR is now marked accepted.
- This changelog and `docs/release-notes/`.

## [0.5.0-alpha] - 2026-04-01

### Fixed

- Partition validation is relaxed to accept legacy schemes ([#18](https://github.com/criteo/command-launcher-registry/pull/18), [#19](https://github.com/criteo/command-launcher-registry/pull/19)).

## [0.4.0-alpha] - 2026-03-26

### Added

- Flexible version format ([#15](https://github.com/criteo/command-launcher-registry/pull/15)).
- Branding package for dynamic app naming ([#16](https://github.com/criteo/command-launcher-registry/pull/16)).

## [0.3.0-alpha] - 2026-02-27

### Added

- Auth detection and JWT token support in the client ([#14](https://github.com/criteo/command-launcher-registry/pull/14)).

## [0.2.0-alpha] - 2026-01-29

### Fixed

- Checksums accept a raw SHA256 hex value ([#13](https://github.com/criteo/command-launcher-registry/pull/13)).

## [0.1.0-alpha] - 2026-01-22

First release.

### Added

- Dynamic version and build information ([#2](https://github.com/criteo/command-launcher-registry/pull/2)).
- LDAP authentication ([#10](https://github.com/criteo/command-launcher-registry/pull/10)).
- Custom JWT authentication ([#12](https://github.com/criteo/command-launcher-registry/pull/12)).
- GitHub Actions build and release workflow, with versioned release binaries ([#3](https://github.com/criteo/command-launcher-registry/pull/3)–[#6](https://github.com/criteo/command-launcher-registry/pull/6), [#8](https://github.com/criteo/command-launcher-registry/pull/8)).
- MIT license and README ([#7](https://github.com/criteo/command-launcher-registry/pull/7), [#9](https://github.com/criteo/command-launcher-registry/pull/9), [#11](https://github.com/criteo/command-launcher-registry/pull/11)).

[0.6.0-alpha]: https://github.com/criteo/command-launcher-registry/compare/0.5.0-alpha...HEAD
[0.5.0-alpha]: https://github.com/criteo/command-launcher-registry/compare/0.4.0-alpha...0.5.0-alpha
[0.4.0-alpha]: https://github.com/criteo/command-launcher-registry/compare/0.3.0-alpha...0.4.0-alpha
[0.3.0-alpha]: https://github.com/criteo/command-launcher-registry/compare/0.2.0-alpha...0.3.0-alpha
[0.2.0-alpha]: https://github.com/criteo/command-launcher-registry/compare/0.1.0-alpha...0.2.0-alpha
[0.1.0-alpha]: https://github.com/criteo/command-launcher-registry/releases/tag/0.1.0-alpha
