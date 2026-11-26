# Helpers for the e2e scenarios. Sourced by run.sh; scenario files use them.
# Bash, runs on the host (macOS or Linux) against the compose stack.

E2E_REGISTRY_PORT="${E2E_REGISTRY_PORT:-28080}"
E2E_MINIO_PORT="${E2E_MINIO_PORT:-29000}"
E2E_URL="${E2E_URL:-http://localhost:${E2E_REGISTRY_PORT}}"
E2E_REGCTL="${E2E_REGCTL:-bin/cdt-regctl}"
E2E_OUT="${E2E_OUT:-bin/e2e}"
E2E_COMPOSE=(docker compose -f tests/e2e/compose.yml)

# A real-looking SHA256 for version payloads.
E2E_CHECKSUM="e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

E2E_PASSED=0
E2E_FAILED=0
E2E_FAILURES=()

# Output of the last run_regctl / http_status call, for assertions.
OUT=""

section() {
	printf '\n== %s\n' "$*"
}

pass() {
	E2E_PASSED=$((E2E_PASSED + 1))
	printf '  ok    %s\n' "$1"
}

fail() {
	E2E_FAILED=$((E2E_FAILED + 1))
	E2E_FAILURES+=("$1")
	printf '  FAIL  %s\n' "$1"
	if [ -n "${2:-}" ]; then
		printf '%s\n' "$2" | sed 's/^/        /'
	fi
}

# check <description> <command...>: pass when the command exits 0.
check() {
	local desc=$1
	shift
	local out
	if out=$("$@" 2>&1); then
		pass "$desc"
	else
		fail "$desc" "$out"
	fi
}

# check_contains <description> <haystack> <needle>
check_contains() {
	if [[ "$2" == *"$3"* ]]; then
		pass "$1"
	else
		fail "$1" "expected to contain: $3"$'\n'"got: $2"
	fi
}

# check_not_contains <description> <haystack> <needle>
check_not_contains() {
	if [[ "$2" != *"$3"* ]]; then
		pass "$1"
	else
		fail "$1" "expected not to contain: $3"$'\n'"got: $2"
	fi
}

b64url() {
	printf '%s' "$1" | base64 | tr -d '\n=' | tr '/+' '_-'
}

# mint_token <user|service> <name> [issuer] [exp-epoch]
# Prints a JWT the stub validator accepts (it does not check signatures).
# Identities and their groups are in tests/e2e/auth/idm-member-of.txt.
mint_token() {
	local kind=$1 name=$2 iss=${3:-criteo-jtc} exp=${4:-$(($(date +%s) + 3600))}
	local claim=ctx:user:uid
	[ "$kind" = service ] && claim=ctx:service:name
	local header='{"alg":"RS256","typ":"JWT","kid":"e2e-stub"}'
	local payload="{\"iss\":\"$iss\",\"$claim\":\"$name\",\"exp\":$exp}"
	printf '%s.%s.e2e-unsigned\n' "$(b64url "$header")" "$(b64url "$payload")"
}

# run_regctl <token> <args...>: runs cdt-regctl against the stack with the
# token in CDT_REGISTRY_TOKEN. Sets OUT to stdout+stderr; returns its status.
run_regctl() {
	local token=$1
	shift
	OUT=$(CDT_REGISTRY_URL="$E2E_URL" CDT_REGISTRY_TOKEN="$token" "$E2E_REGCTL" --yes "$@" 2>&1)
}

# check_regctl <description> <token> <args...>: pass when cdt-regctl exits 0.
check_regctl() {
	local desc=$1
	shift
	if run_regctl "$@"; then
		pass "$desc"
	else
		fail "$desc" "$OUT"
	fi
}

# http_status <method> <path> [token] [json-body]: prints the HTTP status code
# of a raw request. Sets OUT to the response body.
http_status() {
	local method=$1 path=$2 token=${3:-} body=${4:-}
	local args=(-sS -o "$E2E_OUT/body" -w '%{http_code}' -X "$method")
	[ -n "$token" ] && args+=(-H "Authorization: Bearer $token")
	[ -n "$body" ] && args+=(-H 'Content-Type: application/json' -d "$body")
	local code
	code=$(curl "${args[@]}" "$E2E_URL$path") || code=000
	OUT=$(cat "$E2E_OUT/body" 2>/dev/null)
	printf '%s' "$code"
}

# check_status <description> <expected-code> <method> <path> [token] [body]
check_status() {
	local desc=$1 want=$2
	shift 2
	local got
	got=$(http_status "$@")
	OUT=$(cat "$E2E_OUT/body" 2>/dev/null)
	if [ "$got" = "$want" ]; then
		pass "$desc"
	else
		fail "$desc" "expected HTTP $want, got $got: $OUT"
	fi
}

# wait_healthy [seconds]: waits until the server answers its health check.
wait_healthy() {
	local deadline=$(($(date +%s) + ${1:-60}))
	until [ "$(http_status GET /api/v1/health)" = 200 ]; do
		if [ "$(date +%s)" -ge "$deadline" ]; then
			return 1
		fi
		sleep 1
	done
}

# registry_logs: the server container's logs since it last started.
registry_logs() {
	"${E2E_COMPOSE[@]}" logs --no-color --no-log-prefix registry 2>&1
}
