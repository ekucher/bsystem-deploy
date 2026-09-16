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
STAGE_PUBLISH_ADDRESS=127.0.0.1
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

# --- check-hardening.py ------------------------------------------------------
#
# This script is the only thing keeping the container hardening in place:
# Trivy has no Compose rules, and losing a setting breaks nothing at runtime,
# so a regression is invisible until somebody is exploiting it. It renders the
# stacks with Docker, which CI has and a contributor's machine may not, so it
# is exercised here through --rendered against fixtures.

HARD_WORK="$WORK/hardening"
mkdir -p "$HARD_WORK"

# A stack shaped like the real one: hardened, read-only where it should be,
# and publishing only on the loopback address.
write_rendered() {
  cat > "$HARD_WORK/rendered.yml" <<YAML
services:
  postgres:
    security_opt: ["no-new-privileges:true"]
    cap_drop: ["ALL"]
    cap_add: ["CHOWN"]
  authentik-server:
    security_opt: ["no-new-privileges:true"]
  integration-core:
    security_opt: ["no-new-privileges:true"]
    cap_drop: ["ALL"]
    read_only: ${1:-true}
    volumes:
      - type: bind
        source: /repo/postgres/init
        target: /docker-entrypoint-initdb.d
        read_only: ${3:-true}
    ports:
      - host_ip: "${2:-127.0.0.1}"
        published: "8080"
YAML
}

write_rendered
python3 scripts/check-hardening.py --rendered "$HARD_WORK/rendered.yml" >/dev/null 2>&1
check "$?" "0" "a hardened stack passes"

# The property docs/DEPLOYMENT.md states and nothing was checking.
write_rendered false
hard_output="$(python3 scripts/check-hardening.py --rendered "$HARD_WORK/rendered.yml" 2>&1)"
check "$?" "1" "a stateless service losing read_only is caught"
if contains "$hard_output" "integration-core does not run on a read-only root filesystem"; then
  ok "the read-only failure names the service"
else
  bad "the read-only failure does not name the service: $hard_output"
fi

write_rendered true "0.0.0.0"
python3 scripts/check-hardening.py --rendered "$HARD_WORK/rendered.yml" >/dev/null 2>&1
check "$?" "1" "a port published on every interface is caught"

# The failure this script cannot see in its own output: nothing to check reads
# exactly like nothing wrong. A renamed file or an overlay checked alone gets
# here.
printf 'services: {}\n' > "$HARD_WORK/empty.yml"
empty_output="$(python3 scripts/check-hardening.py --rendered "$HARD_WORK/empty.yml" 2>&1)"
check "$?" "1" "a stack that renders no service fails rather than passing vacuously"
if contains "$empty_output" "vacuously"; then
  ok "the empty-stack failure says why an OK would have been wrong"
else
  bad "the empty-stack failure is unclear: $empty_output"
fi

# An exemption that stops being needed must be removed rather than left to rot:
# a silent exemption is how a service comes to look checked.
cat > "$HARD_WORK/exempt.yml" <<'YAML'
services:
  authentik-server:
    security_opt: ["no-new-privileges:true"]
    cap_drop: ["ALL"]
YAML
python3 scripts/check-hardening.py --rendered "$HARD_WORK/exempt.yml" >/dev/null 2>&1
check "$?" "1" "a service that no longer needs its capability exemption is caught"

# A bind mount is a handle on the host filesystem, and every one in these
# stacks carries configuration or seed data inward. They were all :ro before
# anything checked; dropping :ro is a two-character edit that renders and runs.
write_rendered true "127.0.0.1" false
bind_output="$(python3 scripts/check-hardening.py --rendered "$HARD_WORK/rendered.yml" 2>&1)"
check "$?" "1" "a writable bind mount is caught"
if contains "$bind_output" "/repo/postgres/init"; then
  ok "the writable-bind failure names the mount"
else
  bad "the writable-bind failure does not name the mount: $bind_output"
fi

# The same blindness as the empty stack, one level down: a stack with no bind
# mount runs the read-only bind check over nothing, and silence reads exactly
# like a pass.
cat > "$HARD_WORK/nobinds.yml" <<'YAML'
services:
  nats:
    security_opt: ["no-new-privileges:true"]
    cap_drop: ["ALL"]
YAML
nobind_output="$(python3 scripts/check-hardening.py --rendered "$HARD_WORK/nobinds.yml" 2>&1)"
check "$?" "1" "a stack with no bind mount fails rather than passing vacuously"
if contains "$nobind_output" "proves nothing"; then
  ok "the missing-bind failure says why an OK would have been wrong"
else
  bad "the missing-bind failure is unclear: $nobind_output"
fi
# --- documented endpoints exist ----------------------------------------------
#
# An endpoint named in a runbook is followed by somebody at a keyboard. A 404
# from a documented path reads as a broken deployment rather than a stale
# document, so the next twenty minutes go into the deployment.

EP_WORK="$WORK/endpoints/bsystem-deploy"
mkdir -p "$EP_WORK/scripts" "$EP_WORK/docs"
cp scripts/check-documented-endpoints.py "$EP_WORK/scripts/"
ln -sfn "$(cd ../bsystem-integration-core 2>/dev/null && pwd)" "$WORK/endpoints/bsystem-integration-core" 2>/dev/null || true

if [ -f ../bsystem-integration-core/docs/openapi.yaml ]; then
  if python3 scripts/check-documented-endpoints.py >/dev/null 2>&1; then
    ok "every documented endpoint exists in the OpenAPI specification"
  else
    bad "a documented endpoint does not exist: $(python3 scripts/check-documented-endpoints.py 2>&1)"
  fi

  printf 'Check `/api/v1/clientss` to list them.\n' > "$EP_WORK/docs/fake.md"
  ep_output="$(cd "$EP_WORK" && python3 scripts/check-documented-endpoints.py 2>&1)"
  check "$?" "1" "a documented endpoint the platform does not serve is caught"
  if contains "$ep_output" "/api/v1/clientss"; then
    ok "the endpoint failure names the endpoint"
  else
    bad "the endpoint failure does not name the endpoint: $ep_output"
  fi

  # The two shapes that are not endpoints and would otherwise dominate: the
  # versioning rule, and an upstream's own path.
  printf 'Read `/api/v1/clients`. All human routes live under `/api/v1/*`.\n' > "$EP_WORK/docs/fake.md"
  (cd "$EP_WORK" && python3 scripts/check-documented-endpoints.py >/dev/null 2>&1)
  check "$?" "0" "a version prefix written as a rule is not mistaken for an endpoint"

  printf 'Read `/api/v1/clients`. The adapter requests `/api/v1/Account` from EspoCRM.\n' > "$EP_WORK/docs/fake.md"
  (cd "$EP_WORK" && python3 scripts/check-documented-endpoints.py >/dev/null 2>&1)
  check "$?" "0" "an upstream path is not mistaken for a platform endpoint"

  rm -f "$EP_WORK/docs/fake.md"
else
  ok "Integration Core is not checked out; the documented endpoint comparison is skipped"
fi

# --- Global ID prefixes agree with the platform's seed -----------------------
#
# A prefix documented but never seeded allocates nothing: the request fails
# with unsupported_entity_type for a type the rules say exists. A prefix seeded
# but never documented is worse in the other direction, because a Global ID
# that ships is permanent. Nothing else compares the two: the E2E stack
# exercises the types it happens to use.

GID_WORK="$WORK/globalids/bsystem-deploy"
mkdir -p "$GID_WORK/scripts"
cp scripts/check-global-ids.py "$GID_WORK/scripts/"
# The copy is placed so that the sibling layout still holds: the script finds
# Integration Core beside its own repository root, and linking it here exercises
# that resolution rather than a path the test passed in. Without this the
# mutations run against a tree with no Core, where the script correctly reports
# that it compared nothing — and a test that accepts that is testing the skip.
ln -sfn "$(cd ../bsystem-integration-core 2>/dev/null && pwd)" "$WORK/globalids/bsystem-integration-core" 2>/dev/null || true

if [ -d ../bsystem-integration-core/internal/platformdb/migrations ]; then
  if python3 scripts/check-global-ids.py >/dev/null 2>&1; then
    ok "the Global ID prefixes agree with the platform's seed"
  else
    bad "Global ID prefixes disagree: $(python3 scripts/check-global-ids.py 2>&1)"
  fi

  # A prefix in the rules that no migration seeds.
  cp CLAUDE.md "$GID_WORK/CLAUDE.md"
  python3 - "$GID_WORK/CLAUDE.md" <<'PYMUT'
import sys
p = sys.argv[1]
s = open(p).read()
open(p, "w").write(s.replace("REP-*  Repository", "REP-*  Repository\nXYZ-*  Widget", 1))
PYMUT
  gid_output="$(cd "$GID_WORK" && python3 scripts/check-global-ids.py 2>&1)"
  check "$?" "1" "a documented Global ID prefix nothing seeds is caught"
  if contains "$gid_output" "XYZ-"; then
    ok "the prefix failure names the prefix"
  else
    bad "the prefix failure does not name the prefix: $gid_output"
  fi

  # And the other direction: a prefix the platform seeds that nobody wrote down.
  cp CLAUDE.md "$GID_WORK/CLAUDE.md"
  python3 - "$GID_WORK/CLAUDE.md" <<'PYMUT'
import sys
p = sys.argv[1]
s = open(p).read()
open(p, "w").write(s.replace("SVC-*  Service identity\n", "", 1))
PYMUT
  undoc_output="$(cd "$GID_WORK" && python3 scripts/check-global-ids.py 2>&1)"
  check "$?" "1" "a seeded Global ID prefix nobody documented is caught"
  if contains "$undoc_output" "SVC-"; then
    ok "the undocumented-prefix failure names the prefix"
  else
    bad "the undocumented-prefix failure does not name the prefix: $undoc_output"
  fi
else
  ok "Integration Core is not checked out; the Global ID prefix comparison is skipped"
fi

# --- the documentation and the Compose files agree ---------------------------
#
# Three facts live in both places and each drifts in silence. The one that
# prompted this: .env.example documented BIND_ADDRESS, which does nothing in
# stage, and never mentioned STAGE_PUBLISH_ADDRESS, which is what decides
# whether stage is exposed. An operator copying that file could not have known
# the variable existed.

if python3 scripts/check-config-docs.py >/dev/null 2>&1; then
  ok "the documentation and the Compose files agree"
else
  bad "documentation disagrees with the Compose files: $(python3 scripts/check-config-docs.py 2>&1)"
fi

CFG_WORK="$WORK/configdocs"
mkdir -p "$CFG_WORK"

# Each mutation is made on a copy of the tree, never on the tracked file: a
# test that edits what is tracked and puts it back leaves the working tree
# wrong if it is interrupted.
cp -r .env.example docker-compose.yml docker-compose.e2e.yml docker-compose.stage.yml "$CFG_WORK/" 2>/dev/null
mkdir -p "$CFG_WORK/scripts" "$CFG_WORK/docs"
cp scripts/check-config-docs.py "$CFG_WORK/scripts/"
cp docs/RUN-P0.md "$CFG_WORK/docs/"

grep -v 'STAGE_PUBLISH_ADDRESS=127.0.0.1' .env.example > "$CFG_WORK/.env.example"
var_output="$(cd "$CFG_WORK" && python3 scripts/check-config-docs.py 2>&1)"
check "$?" "1" "a variable the stacks read but .env.example omits is caught"
if contains "$var_output" "STAGE_PUBLISH_ADDRESS"; then
  ok "the variable failure names the variable"
else
  bad "the variable failure does not name the variable: $var_output"
fi

cp .env.example "$CFG_WORK/.env.example"
printf '\nOpen `http://localhost:8099/` to continue.\n' >> "$CFG_WORK/docs/RUN-P0.md"
port_output="$(cd "$CFG_WORK" && python3 scripts/check-config-docs.py 2>&1)"
check "$?" "1" "a documented port no stack publishes is caught"
if contains "$port_output" "8099"; then
  ok "the port failure names the port"
else
  bad "the port failure does not name the port: $port_output"
fi

cp docs/RUN-P0.md "$CFG_WORK/docs/RUN-P0.md"
printf '\nRun `docker compose logs integration-kore` to inspect it.\n' >> "$CFG_WORK/docs/RUN-P0.md"
svc_output="$(cd "$CFG_WORK" && python3 scripts/check-config-docs.py 2>&1)"
check "$?" "1" "a documented service no stack defines is caught"
if contains "$svc_output" "integration-kore"; then
  ok "the service failure names the service"
else
  bad "the service failure does not name the service: $svc_output"
fi

# --- every Go module is inside the security scanners -------------------------
#
# The matrix that decides what govulncheck and staticcheck see is written by
# hand, and a module missing from it produces no failure and no output: an
# unscanned module and a clean one print the same nothing. loadtest was in that
# state — compiled by CI, shipped by this repository, scanned by neither.

if python3 scripts/check-scan-coverage.py >/dev/null 2>&1; then
  ok "every Go module is covered by the security scanners"
else
  bad "a Go module is outside the security scanners: $(python3 scripts/check-scan-coverage.py 2>&1)"
fi

SCAN_WORK="$WORK/scan"
mkdir -p "$SCAN_WORK"

# The real workflow is copied rather than edited: a test that mutates a tracked
# file and puts it back leaves the working tree wrong if it is interrupted.
cp .github/workflows/security.yml "$SCAN_WORK/dropped.yml"
sed -i 's/module: \[mocks, e2e, loadtest\]/module: [mocks, e2e]/' "$SCAN_WORK/dropped.yml"
scan_output="$(python3 scripts/check-scan-coverage.py "$SCAN_WORK/dropped.yml" 2>&1)"
check "$?" "1" "a Go module missing from a scanner matrix is caught"
if contains "$scan_output" "loadtest/go.mod"; then
  ok "the coverage failure names the unscanned module"
else
  bad "the coverage failure does not name the module: $scan_output"
fi

# And the other direction: an entry that names nothing scans nothing, and would
# otherwise sit in the matrix looking like coverage.
cp .github/workflows/security.yml "$SCAN_WORK/stale.yml"
sed -i '0,/module: \[mocks, e2e, loadtest\]/s//module: [mocks, e2e, loadtest, ghost]/' "$SCAN_WORK/stale.yml"
stale_output="$(python3 scripts/check-scan-coverage.py "$SCAN_WORK/stale.yml" 2>&1)"
check "$?" "1" "a matrix entry naming no module is caught"
if contains "$stale_output" "ghost"; then
  ok "the stale-entry failure names the entry"
else
  bad "the stale-entry failure does not name the entry: $stale_output"
fi

# --- this repository's Go depends on nothing third-party ---------------------
#
# e2e/harness.go states it as a property of the package: "deliberately has no
# third-party dependencies". Nothing held it. A dependency added here is a
# supply-chain decision and a licence to account for, and the first one arrives
# as a convenience in a test — where it is least likely to be argued about.

for module in mocks e2e loadtest; do
  if [ -f "$module/go.sum" ]; then
    bad "$module has a go.sum: a third-party dependency was added without argument"
  elif grep -qE '^\s*require' "$module/go.mod"; then
    bad "$module/go.mod requires something: a third-party dependency was added"
  else
    ok "$module depends on nothing third-party"
  fi
done

# --- which variable actually publishes the stage ports -----------------------
#
# The stage overlay replaces every port mapping with STAGE_PUBLISH_ADDRESS.
# BIND_ADDRESS governs the base stack and is read by nothing in stage, so a
# preflight that reported on it was answering a question about exposure by
# looking at a variable that does not decide it.

preflight() {
  env -i PATH="$PATH" HOME="$HOME" NO_COLOR=1 ENV_FILE="$1" \
    SKIP_NETWORK=1 SKIP_DOCKER=1 bash scripts/stage-preflight.sh 2>&1
}

write_publish_fixture() {
  cat > "$WORK/env.publish" <<ENV
POSTGRES_PASSWORD=$FIXTURE_DB_PASSWORD
AUTHENTIK_SECRET_KEY=$FIXTURE_AUTHENTIK_KEY
VITE_OIDC_AUTHORITY=https://id.acceptance.invalid/application/o/bsystem-hub/
VITE_OIDC_CLIENT_ID=hub-public-client
$1
ENV
}

# A stage stack published on every interface is the failure this script exists
# to catch, and it was the one configuration it never looked at.
write_publish_fixture "STAGE_PUBLISH_ADDRESS=0.0.0.0"
publish_output="$(preflight "$WORK/env.publish")"
check "$?" "1" "publishing the stage stack on every interface blocks the preflight"
if contains "$publish_output" "published on every interface"; then
  ok "the exposure failure says what is exposed"
else
  bad "the exposure failure is unclear: $publish_output"
fi

# A named interface is a deliberate act rather than a mistake: warn, do not block.
write_publish_fixture "STAGE_PUBLISH_ADDRESS=10.0.0.7"
publish_output="$(preflight "$WORK/env.publish")"
check "$?" "0" "a named interface warns rather than blocking"
if contains "$publish_output" "WARN"; then
  ok "a non-loopback interface is warned about"
else
  bad "a non-loopback interface passed silently: $publish_output"
fi

# Setting the base stack's variable and believing it applies here is the
# mistake the old preflight actively encouraged by reporting on it.
write_publish_fixture "BIND_ADDRESS=0.0.0.0"
publish_output="$(preflight "$WORK/env.publish")"
if contains "$publish_output" "the stage overlay does not read it"; then
  ok "setting BIND_ADDRESS for a stage deployment is called out"
else
  bad "BIND_ADDRESS was accepted as if it governed stage: $publish_output"
fi

# The two runners must agree, or an operator on Windows gets a different answer
# about the same deployment.
for runner in scripts/stage-preflight.sh scripts/stage-preflight.ps1; do
  if grep -q "STAGE_PUBLISH_ADDRESS" "$runner"; then
    ok "$(basename "$runner") checks STAGE_PUBLISH_ADDRESS"
  else
    bad "$(basename "$runner") does not check STAGE_PUBLISH_ADDRESS"
  fi
done

# The acceptance document is what an operator reads to find out which variable
# to set. It named the one that does nothing here.
if grep -q "STAGE_PUBLISH_ADDRESS" docs/STAGE-ACCEPTANCE.md; then
  ok "the stage acceptance table names STAGE_PUBLISH_ADDRESS"
else
  bad "the stage acceptance table does not name STAGE_PUBLISH_ADDRESS"
fi


# --- check-artifacts.py ------------------------------------------------------
#
# bsystem-integration-core carried a 13 MB executable at its root, committed
# because `go build ./cmd/x` writes ./x and .gitignore covered only the
# directories a build can be told to use. This repository has the same shape,
# and had four uncovered mock binaries when the check was first run.

python3 scripts/check-artifacts.py >/dev/null 2>&1
check "$?" "0" "the repository tracks no build artefact"

# A stray binary does not arrive with a helpful suffix, so the check reads
# leading bytes. A .md file holding an ELF header must still fail.
ART_WORK="$WORK/artifacts"
mkdir -p "$ART_WORK"
printf '\177ELF\002\001\001\000' > "notes-from-a-build.md"
git add -f "notes-from-a-build.md" 2>/dev/null
art_output="$(python3 scripts/check-artifacts.py 2>&1)"
art_status=$?
git rm -q --cached "notes-from-a-build.md" 2>/dev/null
rm -f "notes-from-a-build.md"
check "$art_status" "1" "a tracked executable is caught whatever it is named"
if contains "$art_output" "an ELF executable"; then
  ok "the artefact failure says what the file actually is"
else
  bad "the artefact failure does not identify the file: $art_output"
fi

# The second property: the paths a default build writes to must be ignored, so
# committing one is never a \`git add -A\` away. This is the check that found
# the four mock binaries.
for output in mocks/mock-identity mocks/mock-espocrm mocks/mock-redmine mocks/mock-outline loadtest/loadtest; do
  if git check-ignore -q "$output"; then
    ok "git ignores $output"
  else
    bad "git does not ignore $output, which \`go build\` writes"
  fi
done


echo
printf '%d passed, %d failed\n' "$PASSED" "$FAILED"
[ "$FAILED" -gt 0 ] && exit 1
exit 0
