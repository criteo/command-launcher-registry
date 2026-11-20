# Baseline: binary, branding, auth and CRUD through cdt-regctl. Role
# enforcement is on (CDT_REGISTRY_ADMINS in compose.yml), so registry creates
# and deletes run as the server admin, who makes e2e-writer and svc-e2e-ci
# registry admins of the baseline registries so they can write packages and
# versions; tests/e2e/scenarios/20-* and 30-* cover roles.
# Sourced by tests/e2e/run.sh with tests/e2e/lib.sh loaded.

WRITER=$(mint_token user e2e-writer)
SERVER_ADMIN=$(mint_token user e2e-server-admin)
SVC=$(mint_token service svc-e2e-ci)
OUTSIDER=$(mint_token user e2e-outsider)

section "baseline: binary, branding and configuration"

check_status "health check answers on the cdt port (decoy COLA_REGISTRY_SERVER_PORT ignored)" 200 GET /api/v1/health
check_contains "health reports storage ok" "$OUT" '"status":"healthy"'

logs=$(registry_logs)
check_contains "server uses S3 storage on MinIO from CDT_REGISTRY_STORAGE_URI" "$logs" 's3+http://minio:9000/cdt-registry/registry.json'
check_contains "flag --log-level=debug wins over CDT_REGISTRY_LOGGING_LEVEL=warn" "$logs" '"log_level":"debug"'
check_contains "custom_jwt auth is on, set by flag" "$logs" 'Custom JWT authentication enabled'
check_contains "REQUIRED_GROUP comes from CDT_REGISTRY_AUTH_CUSTOM_JWT_REQUIRED_GROUP" "$logs" '"required_group":"g-e2e-writers"'
check_not_contains "storage token is masked in logs" "$logs" 'e2e-secret-key'

run_regctl "" --help
check_contains "cdt-regctl is branded cdt (CDT_REGISTRY_URL in help)" "$OUT" 'CDT_REGISTRY_URL'

section "baseline: authentication (custom_jwt, stub validator, REQUIRED_GROUP)"

check_status "whoami as e2e-writer" 200 GET /api/v1/whoami "$WRITER"
check_contains "whoami returns the username from ctx:user:uid" "$OUT" '"username":"e2e-writer"'
check_contains "whoami returns the groups from the IDM fixture" "$OUT" '"groups":["g-e2e-writers"]'
check_status "whoami as service account svc-e2e-ci" 200 GET /api/v1/whoami "$SVC"
check_contains "whoami returns the service name and groups" "$OUT" '"groups":["g-e2e-writers","g-e2e-ci"]'
check_regctl "cdt-regctl whoami with CDT_REGISTRY_TOKEN" "$WRITER" whoami --json
check_contains "cdt-regctl whoami reports e2e-writer" "$OUT" '"username": "e2e-writer"'

check_status "anonymous write gets 401" 401 POST /api/v1/registry/ "" '{"name":"e2e-anon"}'
check_status "write outside REQUIRED_GROUP gets 401" 401 POST /api/v1/registry/ "$OUTSIDER" '{"name":"e2e-outsider"}'
check_status "whoami outside REQUIRED_GROUP gets 401" 401 GET /api/v1/whoami "$OUTSIDER"
check_status "token from a disallowed issuer gets 401" 401 POST /api/v1/registry/ "$(mint_token user e2e-writer evil-issuer)" '{"name":"e2e-bad-iss"}'
check_status "expired token gets 401" 401 POST /api/v1/registry/ "$(mint_token user e2e-writer criteo-jtc 1000)" '{"name":"e2e-expired"}'
check_status "identity unknown to IDM gets 401" 401 POST /api/v1/registry/ "$(mint_token user e2e-nobody)" '{"name":"e2e-unknown"}'

section "baseline: create and read back through cdt-regctl"

check_regctl "create registry (server admin, writers as registry admins)" "$SERVER_ADMIN" \
	registry create e2e-base --description "E2E baseline registry" --admin "e2e-writer, svc-e2e-ci"
check_regctl "create package" "$WRITER" package create e2e-base hello --description "Hello package"
check_regctl "publish version 1.0.0" "$WRITER" version create e2e-base hello 1.0.0 \
	--checksum "$E2E_CHECKSUM" --url https://example.com/hello-1.0.0.zip --start-partition 0 --end-partition 9
check_regctl "service account creates a package" "$SVC" package create e2e-base from-ci --description "Created by CI"

check_regctl "get registry" "$WRITER" registry get e2e-base --json
check_contains "registry description reads back" "$OUT" 'E2E baseline registry'
check_regctl "list packages" "$WRITER" package list e2e-base --json
check_contains "package hello is listed" "$OUT" '"hello"'
check_contains "package from-ci is listed" "$OUT" '"from-ci"'
check_regctl "get version" "$WRITER" version get e2e-base hello 1.0.0 --json
check_contains "version URL reads back" "$OUT" 'https://example.com/hello-1.0.0.zip'

check_status "anonymous read of index.json" 200 GET /api/v1/registry/e2e-base/index.json
check_contains "index.json lists hello 1.0.0" "$OUT" '"version":"1.0.0"'
check_contains "index.json carries the checksum" "$OUT" "$E2E_CHECKSUM"

section "baseline: delete through cdt-regctl (cascade)"

check_regctl "create throwaway registry (server admin, writer as registry admin)" "$SERVER_ADMIN" \
	registry create e2e-throwaway --admin e2e-writer
check_regctl "create throwaway package" "$WRITER" package create e2e-throwaway tmp
check_regctl "delete throwaway registry (--yes, server admin)" "$SERVER_ADMIN" registry delete e2e-throwaway
check_status "deleted registry is gone" 404 GET /api/v1/registry/e2e-throwaway/

section "baseline: data survives a server restart (S3 persistence)"

"${E2E_COMPOSE[@]}" restart registry >/dev/null 2>&1
if wait_healthy 60; then
	pass "server healthy again after restart"
else
	fail "server healthy again after restart"
fi
check_status "index.json after restart" 200 GET /api/v1/registry/e2e-base/index.json
check_contains "hello 1.0.0 is still in the index" "$OUT" '"version":"1.0.0"'
check_status "throwaway registry is still gone" 404 GET /api/v1/registry/e2e-throwaway/
