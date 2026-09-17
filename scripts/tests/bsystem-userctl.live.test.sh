#!/usr/bin/env bash
#
# Live integration test for scripts/bsystem-userctl.
#
# Requires the local bsystem-platform Compose stack. Creates one dedicated
# temporary authentik user and always removes it.
#
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLI="$ROOT/scripts/bsystem-userctl"
PROJECT="${BSYSTEM_COMPOSE_PROJECT:-bsystem-platform}"

PASSED=0
FAILED=0
SKIPPED=0

ok() {
    printf '  ok    %s\n' "$1"
    PASSED=$((PASSED + 1))
}

bad() {
    printf '  FAIL  %s\n' "$1"
    FAILED=$((FAILED + 1))
}

skip() {
    printf '  skip  %s\n' "$1"
    SKIPPED=$((SKIPPED + 1))
}

contains() {
    case "$1" in
        *"$2"*) return 0 ;;
    esac
    return 1
}

container_for() {
    local service="$1"
    local -a ids=()

    mapfile -t ids < <(
        docker ps \
            --filter "label=com.docker.compose.project=$PROJECT" \
            --filter "label=com.docker.compose.service=$service" \
            --format '{{.ID}}'
    )

    [ "${#ids[@]}" -eq 1 ] || return 1
    printf '%s\n' "${ids[0]}"
}

AK_CONTAINER="$(container_for authentik-worker)" || {
    echo "ERROR: authentik-worker container not found"
    exit 2
}

TEST_USER="userctl-test-$$"
TEST_EMAIL="${TEST_USER}@example.invalid"

PASSWORD1="$(
    python3 - <<'PY'
import secrets
print(secrets.token_urlsafe(24))
PY
)"

PASSWORD2="$(
    python3 - <<'PY'
import secrets
print(secrets.token_urlsafe(24))
PY
)"

cleanup() {
    docker exec \
        -e BSYSTEM_TEST_USERNAME="$TEST_USER" \
        "$AK_CONTAINER" \
        ak shell -c '
import os
from authentik.core.models import User

username = os.environ["BSYSTEM_TEST_USERNAME"]
qs = User.objects.filter(username=username)

if qs.exists():
    qs.delete()
    print(f"cleanup: removed {username}")
else:
    print(f"cleanup: {username} already absent")
' >/dev/null 2>&1 || true

    unset PASSWORD1 PASSWORD2
}

trap cleanup EXIT INT TERM

echo "== bsystem-userctl live integration test =="
echo "temporary user: $TEST_USER"

echo
echo "-- create --"

output="$(
    printf '%s\n%s\n%s\n%s\n%s\n%s\n' \
        "$TEST_USER" \
        "Userctl Live Test" \
        "$TEST_EMAIL" \
        "customer" \
        "$PASSWORD1" \
        "$PASSWORD1" |
        "$CLI" create 2>&1
)"
status=$?

if [ "$status" -eq 0 ] &&
   contains "$output" "User creation: OK" &&
   contains "$output" "BSYSTEM-Customers"; then
    ok "create customer"
else
    bad "create customer"
    printf '%s\n' "$output"
fi

if contains "$output" "$PASSWORD1"; then
    bad "create output leaked password"
else
    ok "create output does not leak password"
fi

echo
echo "-- duplicate create must fail --"

dup_output="$(
    printf '%s\n%s\n%s\n%s\n%s\n%s\n' \
        "$TEST_USER" \
        "Duplicate" \
        "$TEST_EMAIL" \
        "customer" \
        "$PASSWORD1" \
        "$PASSWORD1" |
        "$CLI" create 2>&1
)"
dup_status=$?

if [ "$dup_status" -ne 0 ] &&
   contains "$dup_output" "already exists"; then
    ok "duplicate user refused"
else
    bad "duplicate user was not refused correctly"
    printf '%s\n' "$dup_output"
fi

echo
echo "-- show --"

output="$("$CLI" show "$TEST_USER" 2>&1)"
status=$?

if [ "$status" -eq 0 ] &&
   contains "$output" "$TEST_USER" &&
   contains "$output" "BSYSTEM-Customers"; then
    ok "show customer"
else
    bad "show customer"
    printf '%s\n' "$output"
fi

echo
echo "-- set-role support --"

output="$("$CLI" set-role "$TEST_USER" support 2>&1)"
status=$?

if [ "$status" -eq 0 ] &&
   contains "$output" "BSYSTEM-Support" &&
   contains "$output" "Role update: OK"; then
    ok "customer -> support"
else
    bad "customer -> support"
    printf '%s\n' "$output"
fi

output="$("$CLI" show "$TEST_USER" 2>&1)"

if contains "$output" "BSYSTEM-Support" &&
   ! contains "$output" "BSYSTEM-Customers"; then
    ok "exactly new human role retained"
else
    bad "old human role remained after set-role"
    printf '%s\n' "$output"
fi

echo
echo "-- disable --"

output="$("$CLI" disable "$TEST_USER" 2>&1)"
status=$?

if [ "$status" -eq 0 ] &&
   contains "$output" "Active  : False" &&
   contains "$output" "Disable: OK"; then
    ok "disable user"
else
    bad "disable user"
    printf '%s\n' "$output"
fi

echo
echo "-- enable --"

output="$("$CLI" enable "$TEST_USER" 2>&1)"
status=$?

if [ "$status" -eq 0 ] &&
   contains "$output" "Active  : True" &&
   contains "$output" "Enable: OK"; then
    ok "enable user"
else
    bad "enable user"
    printf '%s\n' "$output"
fi

echo
echo "-- passwd --"

output="$(
    printf '%s\n%s\n' "$PASSWORD2" "$PASSWORD2" |
        "$CLI" passwd "$TEST_USER" 2>&1
)"
status=$?

if [ "$status" -eq 0 ] &&
   contains "$output" "Password: OK" &&
   contains "$output" "Password reset: OK"; then
    ok "password reset"
else
    bad "password reset"
    printf '%s\n' "$output"
fi

if contains "$output" "$PASSWORD2"; then
    bad "passwd output leaked password"
else
    ok "passwd output does not leak password"
fi

echo
echo "-- verify password directly in authentik --"

verify="$(
    printf '%s' "$PASSWORD2" |
        docker exec -i \
            -e BSYSTEM_TEST_USERNAME="$TEST_USER" \
            "$AK_CONTAINER" \
            ak shell -c '
import os
import sys
from authentik.core.models import User

username = os.environ["BSYSTEM_TEST_USERNAME"]
password = sys.stdin.read()
user = User.objects.get(username=username)

print(
    "__PASSWORD_OK__"
    if user.check_password(password)
    else "__PASSWORD_FAILED__"
)
' 2>/dev/null
)"

if contains "$verify" "__PASSWORD_OK__"; then
    ok "authentik accepts new password"
else
    bad "authentik rejected new password"
fi

echo
echo "-- invalid role must fail without mutation --"

output="$("$CLI" set-role "$TEST_USER" superwizard 2>&1)"
status=$?

if [ "$status" -ne 0 ] &&
   contains "$output" "unknown human role"; then
    ok "invalid role refused"
else
    bad "invalid role handling"
    printf '%s\n' "$output"
fi

output="$("$CLI" show "$TEST_USER" 2>&1)"

if contains "$output" "BSYSTEM-Support"; then
    ok "invalid role did not mutate user"
else
    bad "role changed after invalid role attempt"
fi

echo
echo "-- last active administrator protection --"

admin_info="$(
    docker exec "$AK_CONTAINER" \
        ak shell -c '
from authentik.core.models import User

admins = (
    User.objects
    .filter(
        is_active=True,
        groups__name="BSYSTEM-Admins",
    )
    .distinct()
    .order_by("username")
)

print("__ADMIN_COUNT__=" + str(admins.count()))

if admins.count() == 1:
    print("__SOLE_ADMIN__=" + admins.first().username)
' 2>/dev/null
)"

admin_count="$(
    printf '%s\n' "$admin_info" |
        sed -n 's/^__ADMIN_COUNT__=//p' |
        tail -n 1
)"

sole_admin="$(
    printf '%s\n' "$admin_info" |
        sed -n 's/^__SOLE_ADMIN__=//p' |
        tail -n 1
)"

if [ "$admin_count" = "1" ] && [ -n "$sole_admin" ]; then
    output="$("$CLI" disable "$sole_admin" 2>&1)"
    status=$?

    if [ "$status" -ne 0 ] &&
       contains "$output" "last active BSYSTEM administrator"; then
        ok "last active administrator cannot be disabled"
    else
        bad "last active administrator disable protection"
        printf '%s\n' "$output"
    fi

    output="$("$CLI" set-role "$sole_admin" support 2>&1)"
    status=$?

    if [ "$status" -ne 0 ] &&
       contains "$output" "last active BSYSTEM administrator"; then
        ok "last active administrator cannot lose admin role"
    else
        bad "last active administrator role protection"
        printf '%s\n' "$output"
    fi

    output="$("$CLI" show "$sole_admin" 2>&1)"

    if contains "$output" "BSYSTEM-Admins" &&
       ! contains "$output" "BSYSTEM-Support"; then
        ok "failed role change leaves administrator unchanged"
    else
        bad "administrator role changed after refused operation"
        printf '%s\n' "$output"
    fi
else
    skip "last-admin live guard requires exactly one active admin"
fi

echo
echo "-- final temporary-user state --"

output="$("$CLI" show "$TEST_USER" 2>&1)"

if contains "$output" "Active   : True" &&
   contains "$output" "BSYSTEM-Support"; then
    ok "temporary user final state"
else
    bad "unexpected temporary user final state"
    printf '%s\n' "$output"
fi

echo
printf 'RESULT: passed=%d failed=%d skipped=%d\n' \
    "$PASSED" "$FAILED" "$SKIPPED"

if [ "$FAILED" -ne 0 ]; then
    exit 1
fi

exit 0
