# Required-maintainers rule (ticket 12): off by default; with
# CDT_REGISTRY_REQUIRE_MAINTAINERS=true a package cannot be created, or set
# by PUT, with no maintainers, while a package stored with none stays usable.
# Sourced by tests/e2e/run.sh with tests/e2e/lib.sh loaded.

M_SERVER_ADMIN=$(mint_token user e2e-server-admin)

section "require maintainers: off by default"

check_contains "startup log says the rule is off" "$(registry_logs)" \
	'"msg":"Required maintainers rule is off","require_maintainers":false'
check_regctl "server admin creates e2e-reqm" "$M_SERVER_ADMIN" registry create e2e-reqm
check_regctl "rule off: create a package without maintainers" "$M_SERVER_ADMIN" package create e2e-reqm legacy

section "require maintainers: on (CDT_REGISTRY_REQUIRE_MAINTAINERS=true)"

if E2E_REQUIRE_MAINTAINERS=true "${E2E_COMPOSE[@]}" up --detach --wait --no-deps --no-build registry >"$E2E_OUT/reqm-on-up.log" 2>&1 && wait_healthy 60; then
	pass "server recreated with the rule on"
else
	fail "server recreated with the rule on" "$(cat "$E2E_OUT/reqm-on-up.log")"
fi
check_contains "startup log says the rule is on" "$(registry_logs)" \
	'"msg":"Required maintainers rule is on: new packages and package updates need at least one maintainer","require_maintainers":true'

if run_regctl "$M_SERVER_ADMIN" package create e2e-reqm nomaint; then
	fail "create without maintainers fails" "$OUT"
else
	check_contains "create without maintainers fails with the rule's message" "$OUT" "requires at least one maintainer"
fi
check_status "create with empty maintainers: 400" 400 POST /api/v1/registry/e2e-reqm/package "$M_SERVER_ADMIN" '{"name":"nomaint","maintainers":[]}'
check_contains "400 says the server requires a maintainer" "$OUT" "the server requires at least one maintainer"
check_status "no package was created" 404 GET /api/v1/registry/e2e-reqm/package/nomaint

check_regctl "create with a maintainer" "$M_SERVER_ADMIN" package create e2e-reqm owned --maintainer g-e2e-team-a
check_status "PUT maintainers []: 400" 400 PUT /api/v1/registry/e2e-reqm/package/owned "$M_SERVER_ADMIN" '{"maintainers":[]}'
if run_regctl "$M_SERVER_ADMIN" package update e2e-reqm owned --clear-maintainers; then
	fail "update --clear-maintainers fails" "$OUT"
else
	check_contains "update --clear-maintainers fails with the rule's message" "$OUT" "requires at least one maintainer"
fi
check_regctl "update without maintainers succeeds" "$M_SERVER_ADMIN" package update e2e-reqm owned --description "Still owned"
check_regctl "get owned" "$M_SERVER_ADMIN" package get e2e-reqm owned --json
check_contains "owned keeps its maintainer" "$OUT" '"g-e2e-team-a"'
check_contains "owned has the new description" "$OUT" '"Still owned"'

check_regctl "stored package without maintainers: update without maintainers succeeds" "$M_SERVER_ADMIN" \
	package update e2e-reqm legacy --description "Not backfilled yet"
check_regctl "stored package without maintainers: backfill" "$M_SERVER_ADMIN" \
	package update e2e-reqm legacy --maintainer g-e2e-team-a

check_regctl "server admin deletes e2e-reqm" "$M_SERVER_ADMIN" registry delete e2e-reqm

# Back to the default (rule off) for later scenarios.
if "${E2E_COMPOSE[@]}" up --detach --wait --no-deps --no-build registry >"$E2E_OUT/reqm-off-up.log" 2>&1 && wait_healthy 60; then
	pass "server recreated with the rule off"
else
	fail "server recreated with the rule off" "$(cat "$E2E_OUT/reqm-off-up.log")"
fi
check_contains "rule is off again" "$(registry_logs)" '"require_maintainers":false'
