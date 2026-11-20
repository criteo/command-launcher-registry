# Audit lines always visible (ticket 19): with the server at --log-level=warn,
# ordinary Info lines are dropped but every write still logs its audit line.
# Sourced by tests/e2e/run.sh with tests/e2e/lib.sh loaded.

A_SERVER_ADMIN=$(mint_token user e2e-server-admin)
A_WRITER=$(mint_token user e2e-writer)

section "audit: lines visible at --log-level=warn"

if E2E_LOG_LEVEL=warn "${E2E_COMPOSE[@]}" up --detach --wait --no-deps --no-build registry >"$E2E_OUT/audit-warn-up.log" 2>&1 && wait_healthy 60; then
	pass "server recreated with --log-level=warn"
else
	fail "server recreated with --log-level=warn" "$(cat "$E2E_OUT/audit-warn-up.log")"
fi

check_regctl "server admin creates e2e-audit" "$A_SERVER_ADMIN" registry create e2e-audit
check_status "writer without role updates e2e-audit: 403" 403 PUT /api/v1/registry/e2e-audit "$A_WRITER" '{"description":"x"}'
check_regctl "server admin deletes e2e-audit" "$A_SERVER_ADMIN" registry delete e2e-audit

logs=$(registry_logs)
check_not_contains "Info lines are dropped at warn" "$logs" '"msg":"Registry created"'
check_contains "audit line for an allowed write at warn" "$logs" \
	'"level":"INFO","msg":"Write audit","caller":"e2e-server-admin","action":"create_registry","target":"e2e-audit","decision":"allow"'
check_contains "audit line for a denied write at warn" "$logs" \
	'"caller":"e2e-writer","action":"update_registry","target":"e2e-audit","decision":"deny","enforcement":"on","missing_role":"registry admin"'

# Back to the default (--log-level=debug) for later scenarios.
if "${E2E_COMPOSE[@]}" up --detach --wait --no-deps --no-build registry >"$E2E_OUT/audit-debug-up.log" 2>&1 && wait_healthy 60; then
	pass "server recreated with --log-level=debug"
else
	fail "server recreated with --log-level=debug" "$(cat "$E2E_OUT/audit-debug-up.log")"
fi
check_contains "debug level is back" "$(registry_logs)" '"log_level":"debug"'
