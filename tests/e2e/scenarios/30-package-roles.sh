# Package and version roles (ticket 08): create and delete package need
# registry admin; update package (maintainers included) and publish and
# delete version need package maintainer, which registry admins and server
# admins hold; a package with empty maintainers is left to its admins; with
# the server-admin list empty every authenticated write is allowed.
# Identities and groups: tests/e2e/auth/idm-member-of.txt.
# Sourced by tests/e2e/run.sh with tests/e2e/lib.sh loaded.

P_SERVER_ADMIN=$(mint_token user e2e-server-admin)
P_REGISTRY_ADMIN=$(mint_token user e2e-registry-admin)
P_MAINTAINER=$(mint_token user e2e-maintainer) # maintainer of app via group g-e2e-team-a
P_WRITER=$(mint_token user e2e-writer)
P_CI=$(mint_token service svc-e2e-ci) # maintainer of ci-tool only

# publish <token> <package> <version> [extra cdt-regctl args]: cdt-regctl version create.
publish() {
	local token=$1 pkg=$2 version=$3
	run_regctl "$token" version create e2e-pkg "$pkg" "$version" \
		--checksum "$E2E_CHECKSUM" --url "https://example.com/$pkg-$version.zip" --start-partition 0 --end-partition 9
}

# version_body <package> <version>: JSON body for a raw version create.
version_body() {
	printf '{"name":"%s","version":"%s","checksum":"%s","url":"https://example.com/%s-%s.zip","startPartition":0,"endPartition":9}' \
		"$1" "$2" "$E2E_CHECKSUM" "$1" "$2"
}

section "package roles: setup (enforcement on)"

check_regctl "server admin creates e2e-pkg with admin group g-e2e-registry-admins" "$P_SERVER_ADMIN" \
	registry create e2e-pkg --admin g-e2e-registry-admins
check_regctl "server admin creates e2e-pkg-other, admin group g-e2e-team-a" "$P_SERVER_ADMIN" \
	registry create e2e-pkg-other --admin g-e2e-team-a
check_regctl "registry admin creates package app, maintainer group g-e2e-team-a" "$P_REGISTRY_ADMIN" \
	package create e2e-pkg app --maintainer g-e2e-team-a
check_regctl "registry admin creates package ci-tool, maintainer svc-e2e-ci" "$P_REGISTRY_ADMIN" \
	package create e2e-pkg ci-tool --maintainer svc-e2e-ci
check_regctl "registry admin creates package orphan without maintainers" "$P_REGISTRY_ADMIN" \
	package create e2e-pkg orphan

check_status "read back package app" 200 GET /api/v1/registry/e2e-pkg/package/app
check_contains "app maintainers are the given list" "$OUT" '"maintainers":["g-e2e-team-a"]'
check_not_contains "creator is not added to maintainers" "$OUT" 'e2e-registry-admin'

section "package roles: 401 and 403"

check_status "anonymous create package gets 401" 401 POST /api/v1/registry/e2e-pkg/package "" '{"name":"anon"}'
check_status "anonymous publish version gets 401" 401 POST /api/v1/registry/e2e-pkg/package/app/version "" "$(version_body app 0.1.0)"

check_status "writer without role creates a package: 403" 403 POST /api/v1/registry/e2e-pkg/package "$P_WRITER" '{"name":"denied"}'
check_contains "403 has code FORBIDDEN" "$OUT" '"code":"FORBIDDEN"'
check_contains "403 names registry admin on e2e-pkg" "$OUT" '"message":"requires registry admin on e2e-pkg"'
check_status "denied package was not created" 404 GET /api/v1/registry/e2e-pkg/package/denied
check_status "writer without role updates app: 403" 403 PUT /api/v1/registry/e2e-pkg/package/app "$P_WRITER" '{"description":"writer"}'
check_contains "403 names package maintainer on e2e-pkg/app" "$OUT" '"details":{"required_role":"package maintainer","target":"e2e-pkg/app"}'

if publish "$P_WRITER" app 0.1.0; then
	fail "cdt-regctl version create without role fails" "$OUT"
else
	rc=$?
	if [ "$rc" -eq 6 ]; then
		pass "cdt-regctl version create without role exits 6 (permission denied)"
	else
		fail "cdt-regctl version create without role exits 6 (permission denied)" "exit $rc: $OUT"
	fi
fi
check_contains "cdt-regctl prints the server's role message" "$OUT" 'requires package maintainer on e2e-pkg/app'

section "package roles: package maintainer"

if publish "$P_MAINTAINER" app 1.0.0; then pass "maintainer (by group) publishes app 1.0.0"; else fail "maintainer (by group) publishes app 1.0.0" "$OUT"; fi
check_regctl "maintainer updates app description" "$P_MAINTAINER" package update e2e-pkg app --description "By maintainer"
check_regctl "maintainer adds co-maintainer e2e-writer" "$P_MAINTAINER" \
	package update e2e-pkg app --maintainer "g-e2e-team-a, e2e-writer"
if publish "$P_WRITER" app 1.1.0; then pass "co-maintainer publishes app 1.1.0"; else fail "co-maintainer publishes app 1.1.0" "$OUT"; fi
check_regctl "maintainer removes co-maintainer e2e-writer" "$P_MAINTAINER" \
	package update e2e-pkg app --maintainer g-e2e-team-a
check_status "removed co-maintainer publishes: 403" 403 POST /api/v1/registry/e2e-pkg/package/app/version "$P_WRITER" "$(version_body app 1.2.0)"
check_regctl "maintainer deletes version app 1.1.0" "$P_MAINTAINER" version delete e2e-pkg app 1.1.0
check_status "deleted version is gone" 404 GET /api/v1/registry/e2e-pkg/package/app/version/1.1.0

check_status "maintainer creates a package: 403" 403 POST /api/v1/registry/e2e-pkg/package "$P_MAINTAINER" '{"name":"mine"}'
check_status "maintainer deletes own package: 403" 403 DELETE /api/v1/registry/e2e-pkg/package/app "$P_MAINTAINER"
check_contains "403 names registry admin on e2e-pkg" "$OUT" 'requires registry admin on e2e-pkg'
check_status "maintainer of ci-tool publishes to app: 403" 403 POST /api/v1/registry/e2e-pkg/package/app/version "$P_CI" "$(version_body app 1.3.0)"
if publish "$P_CI" ci-tool 1.0.0; then pass "maintainer of ci-tool publishes ci-tool 1.0.0"; else fail "maintainer of ci-tool publishes ci-tool 1.0.0" "$OUT"; fi

section "package roles: empty maintainers, registry admin, server admin"

check_status "maintainer of another package updates orphan: 403" 403 PUT /api/v1/registry/e2e-pkg/package/orphan "$P_MAINTAINER" '{"description":"x"}'
check_status "writer publishes to orphan: 403" 403 POST /api/v1/registry/e2e-pkg/package/orphan/version "$P_WRITER" "$(version_body orphan 1.0.0)"
check_regctl "registry admin updates orphan" "$P_REGISTRY_ADMIN" package update e2e-pkg orphan --description "By registry admin"
if publish "$P_REGISTRY_ADMIN" orphan 1.0.0; then pass "registry admin publishes orphan 1.0.0"; else fail "registry admin publishes orphan 1.0.0" "$OUT"; fi

check_status "registry admin of e2e-pkg-other creates a package in e2e-pkg: 403" 403 \
	POST /api/v1/registry/e2e-pkg/package "$P_MAINTAINER" '{"name":"other-team"}'
check_status "registry admin of e2e-pkg-other deletes ci-tool: 403" 403 DELETE /api/v1/registry/e2e-pkg/package/ci-tool "$P_MAINTAINER"
check_regctl "registry admin of e2e-pkg-other creates a package there" "$P_MAINTAINER" package create e2e-pkg-other own

check_status "server admin publishes to ci-tool" 201 POST /api/v1/registry/e2e-pkg/package/ci-tool/version "$P_SERVER_ADMIN" "$(version_body ci-tool 2.0.0)"
check_regctl "server admin deletes package ci-tool" "$P_SERVER_ADMIN" package delete e2e-pkg ci-tool
check_regctl "registry admin deletes package orphan" "$P_REGISTRY_ADMIN" package delete e2e-pkg orphan
check_status "deleted orphan is gone" 404 GET /api/v1/registry/e2e-pkg/package/orphan

check_status "anonymous read of index.json" 200 GET /api/v1/registry/e2e-pkg/index.json
check_contains "index.json lists app 1.0.0" "$OUT" '"name":"app","version":"1.0.0"'
check_not_contains "index.json carries no maintainers" "$OUT" 'maintainers'
check_not_contains "index.json carries no role principals" "$OUT" 'g-e2e-team-a'

logs=$(registry_logs)
check_contains "audit line for a maintainers change logs old and new lists" "$logs" \
	'"caller":"e2e-maintainer","action":"update_package","target":"e2e-pkg/app","decision":"allow","enforcement":"on","matched_role":"package maintainer","old_maintainers":["g-e2e-team-a"],"new_maintainers":["g-e2e-team-a","e2e-writer"]'
check_contains "audit line for a denied version publish" "$logs" \
	'"caller":"e2e-writer","action":"create_version","target":"e2e-pkg/app/1.2.0","decision":"deny","enforcement":"on","missing_role":"package maintainer"'
check_contains "audit line for a denied package delete" "$logs" \
	'"caller":"e2e-maintainer","action":"delete_package","target":"e2e-pkg/app","decision":"deny","enforcement":"on","missing_role":"registry admin"'

section "package roles: enforcement off (CDT_REGISTRY_ADMINS empty)"

if E2E_ADMINS='' "${E2E_COMPOSE[@]}" up --detach --wait --no-deps --no-build registry >"$E2E_OUT/pkg-roles-off-up.log" 2>&1 && wait_healthy 60; then
	pass "server recreated with empty CDT_REGISTRY_ADMINS"
else
	fail "server recreated with empty CDT_REGISTRY_ADMINS" "$(cat "$E2E_OUT/pkg-roles-off-up.log")"
fi
check_contains "startup log says role enforcement is off" "$(registry_logs)" 'Role enforcement is off'
check_status "anonymous package write still gets 401" 401 POST /api/v1/registry/e2e-pkg/package "" '{"name":"anon"}'
check_regctl "writer without role creates a package" "$P_WRITER" package create e2e-pkg off-pkg
check_regctl "writer without role updates app" "$P_WRITER" package update e2e-pkg app --description "Enforcement off"
if publish "$P_WRITER" app 2.0.0; then pass "writer without role publishes app 2.0.0"; else fail "writer without role publishes app 2.0.0" "$OUT"; fi
check_regctl "writer without role deletes version app 2.0.0" "$P_WRITER" version delete e2e-pkg app 2.0.0
check_regctl "writer without role deletes a package" "$P_WRITER" package delete e2e-pkg off-pkg
check_contains "audit line with enforcement off" "$(registry_logs)" \
	'"caller":"e2e-writer","action":"delete_package","target":"e2e-pkg/off-pkg","decision":"allow","enforcement":"off"'

if "${E2E_COMPOSE[@]}" up --detach --wait --no-deps --no-build registry >"$E2E_OUT/pkg-roles-on-up.log" 2>&1 && wait_healthy 60; then
	pass "server recreated with CDT_REGISTRY_ADMINS=g-e2e-server-admins"
else
	fail "server recreated with CDT_REGISTRY_ADMINS=g-e2e-server-admins" "$(cat "$E2E_OUT/pkg-roles-on-up.log")"
fi
check_status "writer without role is denied again" 403 PUT /api/v1/registry/e2e-pkg/package/app "$P_WRITER" '{"description":"x"}'
check_regctl "server admin deletes e2e-pkg" "$P_SERVER_ADMIN" registry delete e2e-pkg
check_regctl "server admin deletes e2e-pkg-other" "$P_SERVER_ADMIN" registry delete e2e-pkg-other
