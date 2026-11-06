# Command Launcher Registry

A server that hosts registries of Command Launcher packages, and a CLI that manages them.

## Catalog

**Registry**:
A named collection of packages, served to Command Launcher as one `index.json`. One server hosts many registries.

**Package**:
A named command bundle inside one registry.
_Avoid_: Command, tool

**Version**:
One immutable release of a package: its download URL, checksum and partition range.
_Avoid_: Release

## Access

**Role**:
A set of rights a principal holds at one level: the server, one registry, or one package. A role grants every right of the roles below it in the same scope.
_Avoid_: Permission, ACL

**Principal**:
A username or a group name in a role list. It matches a caller whose username or one of whose groups equals it.
_Avoid_: Member, identity

**Server admin**:
The role over the whole server: creates and deletes registries, and holds every registry admin right on every registry. Granted only by server configuration.
_Avoid_: Super admin, global admin

**Registry admin**:
The role over one registry: manages its packages and their versions. Holds every package maintainer right in that registry.
_Avoid_: Registry owner

**Package maintainer**:
The role over one package: publishes and deletes its versions.
_Avoid_: Owner, owner group
