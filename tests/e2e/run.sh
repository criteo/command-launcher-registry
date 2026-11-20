#!/usr/bin/env bash
# Runs the e2e checks against a stack started by `make e2e-up`:
#   1. every tests/e2e/scenarios/*.sh, in name order;
#   2. go test ./tests/integration/... with S3 pointed at the compose MinIO.
# Prints a summary and exits non-zero on any failure (with the server logs).
#
# Adding scenarios: drop a new file in tests/e2e/scenarios/ (NN-name.sh, one
# per ticket, e.g. 20-registry-roles.sh for ticket 07). It is sourced with the
# helpers of tests/e2e/lib.sh loaded: mint_token, run_regctl, check_regctl,
# check_status, check_contains, ...
set -uo pipefail

cd "$(dirname "$0")/../.."
# shellcheck source=lib.sh
source tests/e2e/lib.sh
mkdir -p "$E2E_OUT"

if [ ! -x "$E2E_REGCTL" ]; then
	echo "missing $E2E_REGCTL; run 'make e2e' (it builds cdt-regctl first)" >&2
	exit 2
fi
if ! wait_healthy 60; then
	echo "server at $E2E_URL is not healthy; run 'make e2e-up' first" >&2
	"${E2E_COMPOSE[@]}" ps >&2
	exit 2
fi

for scenario in tests/e2e/scenarios/*.sh; do
	# shellcheck source=/dev/null
	source "$scenario"
done

section "go integration tests (tests/integration, S3 on the compose MinIO)"
integration_log="$E2E_OUT/integration.log"
COLA_TEST_S3_ENDPOINT="localhost:${E2E_MINIO_PORT}" \
	COLA_TEST_S3_BUCKET=integration-tests \
	COLA_TEST_S3_ACCESS_KEY=e2e-access \
	COLA_TEST_S3_SECRET_KEY=e2e-secret-key \
	COLA_TEST_S3_USE_SSL=false \
	go test -count=1 -race -v ./tests/integration/... >"$integration_log" 2>&1
integration_rc=$?
if [ "$integration_rc" -eq 0 ]; then
	pass "go test ./tests/integration/... (log: $integration_log)"
else
	fail "go test ./tests/integration/... (log: $integration_log)" "$(grep -E -e '--- FAIL|^FAIL|Error' "$integration_log" | head -20)"
fi
s3_ran=$(grep -c -e '--- PASS: TestS3' "$integration_log")
s3_skipped=$(grep -c -e '--- SKIP: TestS3' "$integration_log")
if [ "$s3_ran" -gt 0 ] && [ "$s3_skipped" -eq 0 ]; then
	pass "S3 integration tests ran against MinIO ($s3_ran passed, none skipped)"
else
	fail "S3 integration tests ran against MinIO" "passed: $s3_ran, skipped: $s3_skipped"
fi

registry_logs >"$E2E_OUT/registry.log"

printf '\n== summary: %d passed, %d failed\n' "$E2E_PASSED" "$E2E_FAILED"
if [ "$E2E_FAILED" -gt 0 ]; then
	printf '  failed: %s\n' "${E2E_FAILURES[@]}"
	printf '\n== last server log lines (full log: %s)\n' "$E2E_OUT/registry.log"
	tail -n 40 "$E2E_OUT/registry.log"
	exit 1
fi
printf 'e2e green. The stack is still up (%s); stop it with make e2e-down.\n' "$E2E_URL"
