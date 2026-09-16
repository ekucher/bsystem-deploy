#!/usr/bin/env bash
#
# Fixture-based tests for the stage preflight and smoke runner.
#
# They exist because both scripts make a promise that is easy to break by
# accident and expensive to break in practice: neither may print the value of a
# secret. A refactor that starts echoing a variable for debugging would pass
# every other check in this repository.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT" || exit 2

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"; [ -n "${FAKE_PID:-}" ] && kill "$FAKE_PID" 2>/dev/null' EXIT

PASSED=0
FAILED=0

ok()   { printf '  ok    %s\n' "$1"; PASSED=$((PASSED + 1)); }
bad()  { printf '  FAIL  %s\n' "$1"; FAILED=$((FAILED + 1)); }
check() { if [ "$1" = "$2" ]; then ok "$3"; else bad "$3 (got '$1', want '$2')"; fi; }

contains() {
  case "$1" in *"$2"*) return 0 ;; esac
  return 1
}

# Fixture values are generated per run rather than written into this file.
#
# A literal 32-character random-looking string committed to a repository is
# indistinguishable from a real credential, and gitleaks and GitGuardian are
# right to flag it — they cannot know it is a fixture. Generating the values
# keeps the scanners honest instead of teaching them to ignore this file, and
# it makes the leak assertions stronger: each run looks for a string that has
# never existed anywhere before.
random_value() {
  LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c "$1"
}

FIXTURE_DB_PASSWORD="$(random_value 32)"
FIXTURE_AUTHENTIK_KEY="$(random_value 52)"
FIXTURE_HUMAN_TOKEN="$(random_value 24)"
FIXTURE_SERVICE_TOKEN="$(random_value 24)"

# --- preflight --------------------------------------------------------------

echo "== stage-preflight.sh"

cat > "$WORK/env.good" <<ENV
POSTGRES_PASSWORD=$FIXTURE_DB_PASSWORD
AUTHENTIK_SECRET_KEY=$FIXTURE_AUTHENTIK_KEY
VITE_OIDC_AUTHORITY=https://id.acceptance.invalid/application/o/bsystem-hub/
VITE_OIDC_CLIENT_ID=hub-public-client
BIND_ADDRESS=127.0.0.1
ENV

output="$(env -i PATH="$PATH" HOME="$HOME" NO_COLOR=1 ENV_FILE="$WORK/env.good" SKIP_NETWORK=1 SKIP_DOCKER=1 bash scripts/stage-preflight.sh 2>&1)"
check "$?" "0" "a complete configuration passes"

# The whole point of the script's discipline: a secret's value must not appear
# in output that people paste into tickets.
if contains "$output" "$FIXTURE_DB_PASSWORD"; then
  bad "POSTGRES_PASSWORD value leaked into preflight output"
else
  ok "the database password never appears in output"
fi
if contains "$output" "$FIXTURE_AUTHENTIK_KEY"; then
  bad "AUTHENTIK_SECRET_KEY value leaked into preflight output"
else
  ok "the authentik secret key never appears in output"
fi
if contains "$output" "is set (32 characters)"; then
  ok "a secret is reported by length rather than by value"
else
  bad "a secret should be reported by length"
fi

cat > "$WORK/env.placeholder" <<ENV
POSTGRES_PASSWORD=CHANGE_ME_USE_A_LONG_RANDOM_PASSWORD
AUTHENTIK_SECRET_KEY=$FIXTURE_AUTHENTIK_KEY
VITE_OIDC_AUTHORITY=https://id.acceptance.invalid/application/o/bsystem-hub/
VITE_OIDC_CLIENT_ID=hub-public-client
ENV
output="$(env -i PATH="$PATH" HOME="$HOME" NO_COLOR=1 ENV_FILE="$WORK/env.placeholder" SKIP_NETWORK=1 SKIP_DOCKER=1 bash scripts/stage-preflight.sh 2>&1)"
check "$?" "1" "a remaining placeholder blocks the preflight"
if contains "$output" "POSTGRES_PASSWORD still holds a placeholder"; then
  ok "the placeholder is named"
else
  bad "the placeholder should be named"
fi

cat > "$WORK/env.shortkey" <<ENV
POSTGRES_PASSWORD=$FIXTURE_DB_PASSWORD
AUTHENTIK_SECRET_KEY=tooshort
VITE_OIDC_AUTHORITY=https://id.acceptance.invalid/application/o/bsystem-hub/
VITE_OIDC_CLIENT_ID=hub-public-client
ENV
output="$(env -i PATH="$PATH" HOME="$HOME" NO_COLOR=1 ENV_FILE="$WORK/env.shortkey" SKIP_NETWORK=1 SKIP_DOCKER=1 bash scripts/stage-preflight.sh 2>&1)"
check "$?" "1" "a short authentik key blocks the preflight"

cat > "$WORK/env.noslash" <<ENV
POSTGRES_PASSWORD=$FIXTURE_DB_PASSWORD
AUTHENTIK_SECRET_KEY=$FIXTURE_AUTHENTIK_KEY
VITE_OIDC_AUTHORITY=https://id.acceptance.invalid/application/o/bsystem-hub
VITE_OIDC_CLIENT_ID=hub-public-client
ENV
output="$(env -i PATH="$PATH" HOME="$HOME" NO_COLOR=1 ENV_FILE="$WORK/env.noslash" SKIP_NETWORK=1 SKIP_DOCKER=1 bash scripts/stage-preflight.sh 2>&1)"
check "$?" "1" "an issuer without a trailing slash blocks the preflight"

cat > "$WORK/env.outline" <<ENV
POSTGRES_PASSWORD=$FIXTURE_DB_PASSWORD
AUTHENTIK_SECRET_KEY=$FIXTURE_AUTHENTIK_KEY
VITE_OIDC_AUTHORITY=https://id.acceptance.invalid/application/o/bsystem-hub/
VITE_OIDC_CLIENT_ID=hub-public-client
OUTLINE_URL=https://wiki.acceptance.invalid
ENV
output="$(env -i PATH="$PATH" HOME="$HOME" NO_COLOR=1 ENV_FILE="$WORK/env.outline" SKIP_NETWORK=1 SKIP_DOCKER=1 bash scripts/stage-preflight.sh 2>&1)"
check "$?" "1" "an Outline URL with no key blocks the preflight"
if contains "$output" "no anonymous read surface"; then
  ok "the Outline failure explains itself"
else
  bad "the Outline failure should explain itself"
fi

# A .env is untrusted input. Sourcing one would execute it, which is an odd
# thing for a validator to do to a file it is about to call untrusted.
cat > "$WORK/env.hostile" <<ENV
POSTGRES_PASSWORD=$FIXTURE_DB_PASSWORD
AUTHENTIK_SECRET_KEY=$FIXTURE_AUTHENTIK_KEY
VITE_OIDC_AUTHORITY=https://id.acceptance.invalid/application/o/bsystem-hub/
VITE_OIDC_CLIENT_ID=hub-public-client
EVIL=\$(touch /tmp/bsystem-preflight-executed-me)
ENV
rm -f /tmp/bsystem-preflight-executed-me
env -i PATH="$PATH" HOME="$HOME" NO_COLOR=1 ENV_FILE="$WORK/env.hostile" SKIP_NETWORK=1 SKIP_DOCKER=1 bash scripts/stage-preflight.sh >/dev/null 2>&1
if [ -f /tmp/bsystem-preflight-executed-me ]; then
  bad "the preflight executed a command from the .env file"
  rm -f /tmp/bsystem-preflight-executed-me
else
  ok "the .env file is read, not executed"
fi

# --- smoke ------------------------------------------------------------------

echo
echo "== stage-smoke.sh"

python3 - "$WORK" <<'PY' &
import http.server, json, sys, threading, time

class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self, *args): pass
    def do_GET(self):
        path = self.path.split('?')[0]
        authorized = bool(self.headers.get('Authorization'))
        if path == '/health':
            body, code = json.dumps({"status": "ok", "checks": {"database": "ok", "nats": "ok"}}), 200
        elif path == '/readyz':
            body, code = json.dumps({"status": "ready", "checks": {"database": "ok", "nats": "ok"}}), 200
        elif path == '/metrics':
            body, code = 'bsystem_build_info{version="test"} 1\n', 200
        elif path == '/healthz':
            body, code = 'ok', 200
        elif path.startswith('/api/'):
            body, code = (json.dumps({"data": []}), 200) if authorized else (json.dumps({"error": "invalid or expired token"}), 401)
        else:
            body, code = '', 404
        encoded = body.encode()
        self.send_response(code)
        self.send_header('Content-Length', str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

for port in (18080, 18081):
    server = http.server.HTTPServer(('127.0.0.1', port), Handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()
time.sleep(120)
PY
FAKE_PID=$!
sleep 2

SECRET_HUMAN="$FIXTURE_HUMAN_TOKEN"
SECRET_SERVICE="$FIXTURE_SERVICE_TOKEN"

output="$(ARTIFACTS_DIR="$WORK/art" CORE_URL=http://127.0.0.1:18080 HUB_URL=http://127.0.0.1:18081 \
  BSYSTEM_HUMAN_TOKEN="$SECRET_HUMAN" BSYSTEM_SERVICE_TOKEN="$SECRET_SERVICE" TIMEOUT=5 \
  bash scripts/stage-smoke.sh 2>&1)"
smoke_status=$?
check "$smoke_status" "0" "a healthy platform passes the smoke run"

for file in "$WORK/art/stage-acceptance.json" "$WORK/art/stage-acceptance.md"; do
  if [ -f "$file" ]; then ok "$(basename "$file") was written"; else bad "$(basename "$file") is missing"; fi
done

if python3 -c "import json,sys; json.load(open(sys.argv[1]))" "$WORK/art/stage-acceptance.json" 2>/dev/null; then
  ok "the JSON report parses"
else
  bad "the JSON report does not parse"
fi

# The reason this file exists.
for where in "$output" "$(cat "$WORK/art/stage-acceptance.json" 2>/dev/null)" "$(cat "$WORK/art/stage-acceptance.md" 2>/dev/null)"; do
  if contains "$where" "$SECRET_HUMAN" || contains "$where" "$SECRET_SERVICE"; then
    bad "a token leaked into smoke output or its report"
    leaked=1
  fi
done
[ -z "${leaked:-}" ] && ok "no token appears in the output or either report"

# An unauthenticated caller reaching the API is the finding that should stop an
# acceptance, so the runner has to check for it rather than assume it.
if contains "$output" "rejects anonymous"; then
  ok "the run asserts that anonymous callers are rejected"
else
  bad "the run should assert that anonymous callers are rejected"
fi

if contains "$output" "BLOCK"; then
  ok "owner-blocked items are reported as BLOCKED rather than passed"
else
  bad "owner-blocked items should be reported"
fi

# Without a service token the machine API must be skipped, not failed, and not
# reached some other way.
output="$(ARTIFACTS_DIR="$WORK/art2" CORE_URL=http://127.0.0.1:18080 HUB_URL=http://127.0.0.1:18081 \
  BSYSTEM_HUMAN_TOKEN="$SECRET_HUMAN" TIMEOUT=5 bash scripts/stage-smoke.sh 2>&1)"
check "$?" "0" "a missing service token does not fail the run"
if contains "$output" "SKIP   integration-core       GET /api/service/v1/whoami"; then
  ok "machine API checks are skipped with a reason"
else
  bad "machine API checks should be skipped with a reason"
fi

output="$(ARTIFACTS_DIR="$WORK/art3" CORE_URL=http://127.0.0.1:59999 HUB_URL=http://127.0.0.1:59998 TIMEOUT=2 \
  bash scripts/stage-smoke.sh 2>&1)"
check "$?" "1" "an unreachable platform fails the run"

# --- release manifest -------------------------------------------------------

echo
echo "== release-manifest.sh"

if ./scripts/release-manifest.sh | python3 -c 'import json,sys; json.load(sys.stdin)' 2>/dev/null; then
  ok "the manifest is valid JSON with the sibling repositories present"
else
  bad "the manifest is not valid JSON with the sibling repositories present"
fi

# CI checks out this repository alone, so every sibling is absent there. The
# first CI run produced `"dirty": unavailable` — a bare token in a boolean
# field — while the local run passed, because locally the siblings exist.
if CORE_DIR=/nonexistent HUB_DIR=/nonexistent DS_DIR=/nonexistent \
   ./scripts/release-manifest.sh | python3 -c 'import json,sys; json.load(sys.stdin)' 2>/dev/null; then
  ok "the manifest is valid JSON with every sibling repository absent"
else
  bad "the manifest is not valid JSON with every sibling repository absent"
fi

# --- check-identity-groups.py ------------------------------------------------
#
# The checker compares three lists of BSYSTEM group names. A checker that
# silently matches nothing would report agreement between two empty sets, which
# is the failure it exists to prevent — and the first draft did exactly that in
# reverse, capturing "name: BSYSTEM-Admins" instead of the group, so every name
# looked like a mismatch. Both directions are covered here.

./scripts/check-identity-groups.py >/dev/null 2>&1
check "$?" "0" "the group names agree as committed"

GROUPS_WORK="$WORK/groups"
mkdir -p "$GROUPS_WORK/scripts" "$GROUPS_WORK/authentik/blueprints" \
         "$GROUPS_WORK/mocks/cmd/mock-identity"
cp scripts/check-identity-groups.py "$GROUPS_WORK/scripts/"

write_group_fixtures() {
  printf 'entries:\n  - identifiers:\n      name: %s\n' "$1" \
    > "$GROUPS_WORK/authentik/blueprints/bsystem-groups.yaml"
  printf 'package main\nvar x = Principal{Groups: []string{"%s"}}\n' "$2" \
    > "$GROUPS_WORK/mocks/cmd/mock-identity/principals.go"
}

write_group_fixtures "BSYSTEM-Admins" "BSYSTEM-Admins"
(cd "$GROUPS_WORK" && ./scripts/check-identity-groups.py >/dev/null 2>&1)
check "$?" "0" "matching group names pass"

# The real defect: one character, and the platform silently grants nothing.
write_group_fixtures "BSYSTEM-Admin" "BSYSTEM-Admins"
group_output="$(cd "$GROUPS_WORK" && ./scripts/check-identity-groups.py 2>&1)"
check "$?" "1" "a blueprint typo is caught"
if contains "$group_output" "BSYSTEM-Admin "; then
  ok "the mismatch names the group rather than the line it was found on"
else
  bad "the mismatch does not name the group cleanly: $group_output"
fi

# A blueprint with no groups must not read as "everything agrees".
printf 'entries: []\n' > "$GROUPS_WORK/authentik/blueprints/bsystem-groups.yaml"
(cd "$GROUPS_WORK" && ./scripts/check-identity-groups.py >/dev/null 2>&1)
check "$?" "1" "an empty blueprint fails rather than matching an empty set"

echo
printf '%d passed, %d failed\n' "$PASSED" "$FAILED"
[ "$FAILED" -gt 0 ] && exit 1
exit 0
