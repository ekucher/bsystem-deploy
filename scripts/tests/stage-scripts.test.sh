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

# --- preflight --------------------------------------------------------------

echo "== stage-preflight.sh"

cat > "$WORK/env.good" <<'ENV'
POSTGRES_PASSWORD=g7Qx2vLp9SdKmR4tYbN8wZ3aFcHjUe6X
AUTHENTIK_SECRET_KEY=Ku3pS9vLmQ2xTz8RbY4nWd6FgHjKlPoIuYtReWqAsDfGhJkLzX
VITE_OIDC_AUTHORITY=https://id.acceptance.invalid/application/o/bsystem-hub/
VITE_OIDC_CLIENT_ID=hub-public-client
BIND_ADDRESS=127.0.0.1
ENV

output="$(env -i PATH="$PATH" HOME="$HOME" NO_COLOR=1 ENV_FILE="$WORK/env.good" SKIP_NETWORK=1 SKIP_DOCKER=1 bash scripts/stage-preflight.sh 2>&1)"
check "$?" "0" "a complete configuration passes"

# The whole point of the script's discipline: a secret's value must not appear
# in output that people paste into tickets.
if contains "$output" "g7Qx2vLp9SdKmR4tYbN8wZ3aFcHjUe6X"; then
  bad "POSTGRES_PASSWORD value leaked into preflight output"
else
  ok "the database password never appears in output"
fi
if contains "$output" "Ku3pS9vLmQ2xTz8RbY4nWd6FgHjKlPoIuYtReWqAsDfGhJkLzX"; then
  bad "AUTHENTIK_SECRET_KEY value leaked into preflight output"
else
  ok "the authentik secret key never appears in output"
fi
if contains "$output" "is set (32 characters)"; then
  ok "a secret is reported by length rather than by value"
else
  bad "a secret should be reported by length"
fi

cat > "$WORK/env.placeholder" <<'ENV'
POSTGRES_PASSWORD=CHANGE_ME_USE_A_LONG_RANDOM_PASSWORD
AUTHENTIK_SECRET_KEY=Ku3pS9vLmQ2xTz8RbY4nWd6FgHjKlPoIuYtReWqAsDfGhJkLzX
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

cat > "$WORK/env.shortkey" <<'ENV'
POSTGRES_PASSWORD=g7Qx2vLp9SdKmR4tYbN8wZ3aFcHjUe6X
AUTHENTIK_SECRET_KEY=tooshort
VITE_OIDC_AUTHORITY=https://id.acceptance.invalid/application/o/bsystem-hub/
VITE_OIDC_CLIENT_ID=hub-public-client
ENV
output="$(env -i PATH="$PATH" HOME="$HOME" NO_COLOR=1 ENV_FILE="$WORK/env.shortkey" SKIP_NETWORK=1 SKIP_DOCKER=1 bash scripts/stage-preflight.sh 2>&1)"
check "$?" "1" "a short authentik key blocks the preflight"

cat > "$WORK/env.noslash" <<'ENV'
POSTGRES_PASSWORD=g7Qx2vLp9SdKmR4tYbN8wZ3aFcHjUe6X
AUTHENTIK_SECRET_KEY=Ku3pS9vLmQ2xTz8RbY4nWd6FgHjKlPoIuYtReWqAsDfGhJkLzX
VITE_OIDC_AUTHORITY=https://id.acceptance.invalid/application/o/bsystem-hub
VITE_OIDC_CLIENT_ID=hub-public-client
ENV
output="$(env -i PATH="$PATH" HOME="$HOME" NO_COLOR=1 ENV_FILE="$WORK/env.noslash" SKIP_NETWORK=1 SKIP_DOCKER=1 bash scripts/stage-preflight.sh 2>&1)"
check "$?" "1" "an issuer without a trailing slash blocks the preflight"

cat > "$WORK/env.outline" <<'ENV'
POSTGRES_PASSWORD=g7Qx2vLp9SdKmR4tYbN8wZ3aFcHjUe6X
AUTHENTIK_SECRET_KEY=Ku3pS9vLmQ2xTz8RbY4nWd6FgHjKlPoIuYtReWqAsDfGhJkLzX
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
cat > "$WORK/env.hostile" <<'ENV'
POSTGRES_PASSWORD=g7Qx2vLp9SdKmR4tYbN8wZ3aFcHjUe6X
AUTHENTIK_SECRET_KEY=Ku3pS9vLmQ2xTz8RbY4nWd6FgHjKlPoIuYtReWqAsDfGhJkLzX
VITE_OIDC_AUTHORITY=https://id.acceptance.invalid/application/o/bsystem-hub/
VITE_OIDC_CLIENT_ID=hub-public-client
EVIL=$(touch /tmp/bsystem-preflight-executed-me)
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

SECRET_HUMAN="human-token-3f9a2b7c1e"
SECRET_SERVICE="service-token-8d4e6a0b5c"

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

echo
printf '%d passed, %d failed\n' "$PASSED" "$FAILED"
[ "$FAILED" -gt 0 ] && exit 1
exit 0
