#!/bin/sh
# E2E stand-in for the deployed custom_jwt validator
# (containers/criteo-images, images/10-criteo-cdt-registry/files/verify_jwt.py).
#
# Same contract as the deployed script, without its two network calls:
# - same token checks except the signature: three parts, alg RS256 with a kid,
#   iss in JWT_ALLOWED_ISSUERS, exp not in the past;
# - identity from the claim ctx:service:name (service account) or
#   ctx:user:uid (human), service first;
# - groups from a fixture file standing in for the IDM API's member_of field.
#   An identity missing from the fixture fails, as an IDM miss does.
#
# Output (custom_jwt format): "username:<id>", then one group per line.
# Any failure exits 1 with the reason on stderr.
set -eu

IDM_FILE="${E2E_IDM_FILE:-$(dirname "$0")/idm-member-of.txt}"
ALLOWED_ISSUERS="${JWT_ALLOWED_ISSUERS:-criteo-jtc,criteo-jenkins}"

fail() {
	echo "$*" >&2
	exit 1
}

[ "$#" -eq 1 ] || fail "Usage: $0 <jwt_token>"
token=$1

[ "$(printf '%s' "$token" | tr -cd '.' | wc -c | tr -d ' ')" = 2 ] || fail "Invalid JWT format"

b64url_decode() {
	s=$(printf '%s' "$1" | tr '_-' '/+')
	case $((${#s} % 4)) in
	2) s="$s==" ;;
	3) s="$s=" ;;
	1) return 1 ;;
	esac
	printf '%s' "$s" | base64 -d 2>/dev/null
}

header=$(b64url_decode "$(printf '%s' "$token" | cut -d. -f1)") || fail "Invalid JWT: bad header encoding"
payload=$(b64url_decode "$(printf '%s' "$token" | cut -d. -f2)") || fail "Invalid JWT: bad payload encoding"

# str_field <json> <key>: value of a string field in compact, flat JSON.
str_field() {
	printf '%s' "$1" | sed -n "s/.*\"$2\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p"
}
# num_field <json> <key>: value of an integer field in compact, flat JSON.
num_field() {
	printf '%s' "$1" | sed -n "s/.*\"$2\"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p"
}

alg=$(str_field "$header" alg)
[ "$alg" = RS256 ] || fail "Unsupported algorithm: $alg (only RS256 supported)"
[ -n "$(str_field "$header" kid)" ] || fail "Missing 'kid' in JWT header"

iss=$(str_field "$payload" iss)
[ -n "$iss" ] || fail "Missing 'iss' claim in JWT"
case ",$ALLOWED_ISSUERS," in
*",$iss,"*) ;;
*) fail "Issuer '$iss' is not allowed" ;;
esac

exp=$(num_field "$payload" exp)
if [ -n "$exp" ] && [ "$(date +%s)" -gt "$exp" ]; then
	fail "Token has expired"
fi

service=$(str_field "$payload" 'ctx:service:name')
user=$(str_field "$payload" 'ctx:user:uid')
if [ -n "$service" ]; then
	kind=service
	id=$service
elif [ -n "$user" ]; then
	kind=user
	id=$user
else
	fail "Missing identity claim: expected 'ctx:user:uid' or 'ctx:service:name'"
fi

groups=$(awk -v kind="$kind" -v id="$id" '
	$1 == kind && $2 == id { found = 1; for (i = 3; i <= NF; i++) print $i }
	END { exit found ? 0 : 1 }
' "$IDM_FILE") || fail "$kind '$id' not found in IDM fixture"

echo "username:$id"
[ -z "$groups" ] || printf '%s\n' "$groups"
