# Registry roles (ticket 07): create/delete registry need server admin,
# update needs registry admin of that registry; 401 stays for anonymous
# writes; with the server-admin list empty every authenticated write is
# allowed. Identities and groups: tests/e2e/auth/idm-member-of.txt.
# Sourced by tests/e2e/run.sh with tests/e2e/lib.sh loaded.

R_SERVER_ADMIN=$(mint_token user e2e-server-admin)
R_REGISTRY_ADMIN=$(mint_token user e2e-registry-admin)
R_COADMIN=$(mint_token user e2e-maintainer)
R_WRITER=$(mint_token user e2e-writer)

section "registry roles: enforcement on (CDT_REGISTRY_ADMINS=g-e2e-server-admins)"

logs=$(registry_logs)
check_contains "startup log says role enforcement is on" "$logs" 'Role enforcement is on'

check_status "whoami as server admin (by group)" 200 GET /api/v1/whoami "$R_SERVER_ADMIN"
check_contains "whoami reports server_admin true" "$OUT" '"server_admin":true'
check_status "whoami as writer without role" 200 GET /api/v1/whoami "$R_WRITER"
check_contains "whoami reports server_admin false" "$OUT" '"server_admin":false'

check_regctl "server admin creates e2e-roles with admin group g-e2e-registry-admins" "$R_SERVER_ADMIN" \
	registry create e2e-roles --description "Roles registry" --admin g-e2e-registry-admins
check_regctl "server admin creates e2e-roles-other (another team's registry)" "$R_SERVER_ADMIN" \
	registry create e2e-roles-other --admin g-e2e-team-a

# 401: unauthenticated writes keep their 401.
check_status "anonymous update registry gets 401" 401 PUT /api/v1/registry/e2e-roles "" '{"description":"anon"}'
check_status "anonymous delete registry gets 401" 401 DELETE /api/v1/registry/e2e-roles

# 403: authenticated caller without the role.
check_status "writer without role creates a registry: 403" 403 POST /api/v1/registry/ "$R_WRITER" '{"name":"e2e-roles-denied"}'
check_contains "403 has code FORBIDDEN" "$OUT" '"code":"FORBIDDEN"'
check_contains "403 names the missing role and target" "$OUT" '"message":"requires server admin on e2e-roles-denied"'
check_contains "403 details carry required_role and target" "$OUT" '"details":{"required_role":"server admin","target":"e2e-roles-denied"}'
check_status "denied registry was not created" 404 GET /api/v1/registry/e2e-roles-denied/
check_status "writer without role updates e2e-roles: 403" 403 PUT /api/v1/registry/e2e-roles "$R_WRITER" '{"description":"writer"}'
check_contains "403 names registry admin on e2e-roles" "$OUT" 'requires registry admin on e2e-roles'
check_status "writer without role deletes e2e-roles: 403" 403 DELETE /api/v1/registry/e2e-roles "$R_WRITER"

if run_regctl "$R_WRITER" registry delete e2e-roles; then
	fail "cdt-regctl registry delete without role fails" "$OUT"
else
	rc=$?
	if [ "$rc" -eq 6 ]; then
		pass "cdt-regctl registry delete without role exits 6 (permission denied)"
	else
		fail "cdt-regctl registry delete without role exits 6 (permission denied)" "exit $rc: $OUT"
	fi
fi
check_contains "cdt-regctl prints the server's role message" "$OUT" 'requires server admin on e2e-roles'

# Registry admin: own registry only, and no create/delete.
check_regctl "registry admin (by group) updates e2e-roles" "$R_REGISTRY_ADMIN" \
	registry update e2e-roles --description "Updated by registry admin"
check_regctl "read back e2e-roles" "$R_WRITER" registry get e2e-roles --json
check_contains "registry admin's description is stored" "$OUT" 'Updated by registry admin'
check_status "registry admin updates another registry: 403" 403 PUT /api/v1/registry/e2e-roles-other "$R_REGISTRY_ADMIN" '{"description":"x"}'
check_contains "403 names registry admin on e2e-roles-other" "$OUT" 'requires registry admin on e2e-roles-other'
check_status "registry admin creates a registry: 403" 403 POST /api/v1/registry/ "$R_REGISTRY_ADMIN" '{"name":"e2e-roles-denied"}'
check_status "registry admin deletes own registry: 403" 403 DELETE /api/v1/registry/e2e-roles "$R_REGISTRY_ADMIN"

# Registry admins manage co-admins.
check_regctl "registry admin adds co-admin e2e-maintainer" "$R_REGISTRY_ADMIN" \
	registry update e2e-roles --admin "g-e2e-registry-admins, e2e-maintainer"
check_status "co-admin updates e2e-roles" 200 PUT /api/v1/registry/e2e-roles "$R_COADMIN" '{"description":"Updated by co-admin"}'
check_regctl "registry admin removes co-admin e2e-maintainer" "$R_REGISTRY_ADMIN" \
	registry update e2e-roles --admin g-e2e-registry-admins
check_status "removed co-admin updates e2e-roles: 403" 403 PUT /api/v1/registry/e2e-roles "$R_COADMIN" '{"description":"after removal"}'

# Server admin holds registry admin everywhere and deletes registries.
check_status "server admin updates another team's registry" 200 PUT /api/v1/registry/e2e-roles-other "$R_SERVER_ADMIN" '{"description":"fixed by server admin"}'
check_regctl "server admin deletes e2e-roles-other" "$R_SERVER_ADMIN" registry delete e2e-roles-other
check_status "deleted e2e-roles-other is gone" 404 GET /api/v1/registry/e2e-roles-other/

# Audit lines on stdout.
logs=$(registry_logs)
check_contains "audit line for a denied write" "$logs" \
	'"caller":"e2e-writer","action":"create_registry","target":"e2e-roles-denied","decision":"deny","enforcement":"on","missing_role":"server admin"'
check_contains "audit line for an allowed write" "$logs" \
	'"caller":"e2e-server-admin","action":"delete_registry","target":"e2e-roles-other","decision":"allow","enforcement":"on","matched_role":"server admin"'
check_contains "audit line for an admins change logs old and new lists" "$logs" \
	'"old_admins":["g-e2e-registry-admins"],"new_admins":["g-e2e-registry-admins","e2e-maintainer"]'
check_not_contains "no token in the logs" "$logs" "$R_WRITER"

section "registry roles: enforcement off (CDT_REGISTRY_ADMINS empty)"

# Recreate only the server with an empty server-admin list; MinIO keeps
# the data.
if E2E_ADMINS='' "${E2E_COMPOSE[@]}" up --detach --wait --no-deps --no-build registry >"$E2E_OUT/roles-off-up.log" 2>&1 && wait_healthy 60; then
	pass "server recreated with empty CDT_REGISTRY_ADMINS"
else
	fail "server recreated with empty CDT_REGISTRY_ADMINS" "$(cat "$E2E_OUT/roles-off-up.log")"
fi
logs=$(registry_logs)
check_contains "startup log says role enforcement is off" "$logs" 'Role enforcement is off'

check_status "whoami as server admin reports server_admin false" 200 GET /api/v1/whoami "$R_SERVER_ADMIN"
check_contains "server_admin is false with enforcement off" "$OUT" '"server_admin":false'
check_status "anonymous write still gets 401" 401 POST /api/v1/registry/ "" '{"name":"e2e-roles-off"}'
check_regctl "writer without role creates a registry" "$R_WRITER" registry create e2e-roles-off
check_regctl "writer without role updates a registry it does not admin" "$R_WRITER" \
	registry update e2e-roles --description "Updated with enforcement off"
check_regctl "writer without role deletes a registry" "$R_WRITER" registry delete e2e-roles-off
logs=$(registry_logs)
check_contains "audit line with enforcement off" "$logs" \
	'"caller":"e2e-writer","action":"delete_registry","target":"e2e-roles-off","decision":"allow","enforcement":"off"'

# Back to the default (enforcement on) for later scenarios.
if "${E2E_COMPOSE[@]}" up --detach --wait --no-deps --no-build registry >"$E2E_OUT/roles-on-up.log" 2>&1 && wait_healthy 60; then
	pass "server recreated with CDT_REGISTRY_ADMINS=g-e2e-server-admins"
else
	fail "server recreated with CDT_REGISTRY_ADMINS=g-e2e-server-admins" "$(cat "$E2E_OUT/roles-on-up.log")"
fi
check_contains "role enforcement is on again" "$(registry_logs)" 'Role enforcement is on'
check_status "writer without role is denied again" 403 DELETE /api/v1/registry/e2e-roles "$R_WRITER"
check_regctl "server admin deletes e2e-roles" "$R_SERVER_ADMIN" registry delete e2e-roles
