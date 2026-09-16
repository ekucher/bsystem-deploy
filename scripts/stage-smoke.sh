#!/usr/bin/env bash
#
# Stage smoke test: a non-destructive acceptance pass over a running stage
# deployment.
#
# Every check is a GET. Nothing is created, updated or deleted, so this can be
# run against an environment someone else is using without asking first.
#
# Tokens are read from the environment and never printed, never written to the
# report, and never included in a diagnostic. A check that needs a token it
# does not have is SKIPped with the reason — the alternative, failing the whole
# unauthenticated run because no service token was available, would train
# people to ignore the result.
#
# Outputs:
#   artifacts/stage-acceptance.json   machine-readable
#   artifacts/stage-acceptance.md     human-readable
#
# Exit codes:
#   0  no check failed
#   1  at least one check failed
#   2  the runner could not start
set -uo pipefail

CORE_URL="${CORE_URL:-http://127.0.0.1:8080}"
HUB_URL="${HUB_URL:-http://127.0.0.1:8081}"
AUTHENTIK_URL="${AUTHENTIK_URL:-}"
ARTIFACTS_DIR="${ARTIFACTS_DIR:-artifacts}"
TIMEOUT="${TIMEOUT:-10}"

# Tokens are optional by design. Their absence downgrades checks to SKIP.
HUMAN_TOKEN="${BSYSTEM_HUMAN_TOKEN:-}"
SERVICE_TOKEN="${BSYSTEM_SERVICE_TOKEN:-}"

command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 2; }

mkdir -p "$ARTIFACTS_DIR" || exit 2

STARTED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
RESULTS_FILE="$(mktemp)"
trap 'rm -f "$RESULTS_FILE"' EXIT

FAILED=0
PASSED=0
SKIPPED=0
BLOCKED=0

json_escape() {
  # Escape the characters JSON forbids raw. Diagnostics are ours rather than
  # arbitrary upstream text, but a report that cannot be parsed is useless
  # exactly when it matters.
  printf '%s' "$1" | python3 -c 'import json,sys; sys.stdout.write(json.dumps(sys.stdin.read())[1:-1])' 2>/dev/null \
    || printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g'
}

record() {
  local service="$1" check="$2" result="$3" duration_ms="$4" request_id="$5" message="$6"
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$service" "$check" "$result" "$duration_ms" "$request_id" "$message" >> "$RESULTS_FILE"
  case "$result" in
    PASS) PASSED=$((PASSED + 1)); printf '  PASS   %-22s %s\n' "$service" "$check" ;;
    FAIL) FAILED=$((FAILED + 1)); printf '  FAIL   %-22s %s — %s\n' "$service" "$check" "$message" ;;
    SKIP) SKIPPED=$((SKIPPED + 1)); printf '  SKIP   %-22s %s — %s\n' "$service" "$check" "$message" ;;
    BLOCKED) BLOCKED=$((BLOCKED + 1)); printf '  BLOCK  %-22s %s — %s\n' "$service" "$check" "$message" ;;
  esac
}

# http_check performs one GET and records the outcome.
#
# The correlation id is generated here and sent, so a failed check can be found
# in the platform's logs. It is not a secret and appears in the report on
# purpose: pairing a red row with a log line is the whole point of having one.
http_check() {
  local service="$1" check="$2" url="$3" expected="$4" auth_kind="${5:-none}"
  local token=''
  case "$auth_kind" in
    human)
      token="$HUMAN_TOKEN"
      if [ -z "$token" ]; then
        record "$service" "$check" SKIP 0 "" "no human token available (set BSYSTEM_HUMAN_TOKEN)"
        return
      fi ;;
    service)
      token="$SERVICE_TOKEN"
      if [ -z "$token" ]; then
        record "$service" "$check" SKIP 0 "" "no service token available (set BSYSTEM_SERVICE_TOKEN); the machine API is not reachable with a human token by design"
        return
      fi ;;
  esac

  local request_id
  request_id="smoke-$(date -u +%H%M%S)-$RANDOM"

  local started ended duration status
  started="$(date +%s%3N 2>/dev/null || echo 0)"

  local -a args=(--silent --show-error --output /dev/null --write-out '%{http_code}'
                 --max-time "$TIMEOUT" --header "X-Request-ID: $request_id")
  if [ -n "$token" ]; then
    args+=(--header "Authorization: Bearer $token")
  fi

  status="$(curl "${args[@]}" "$url" 2>/dev/null)"
  ended="$(date +%s%3N 2>/dev/null || echo 0)"
  duration=$((ended - started))
  [ "$duration" -lt 0 ] && duration=0

  if [ -z "$status" ] || [ "$status" = "000" ]; then
    record "$service" "$check" FAIL "$duration" "$request_id" "no response from $url within ${TIMEOUT}s"
    return
  fi

  case " $expected " in
    *" $status "*) record "$service" "$check" PASS "$duration" "$request_id" "HTTP $status" ;;
    *) record "$service" "$check" FAIL "$duration" "$request_id" "HTTP $status, expected one of: $expected" ;;
  esac
}

echo "BSYSTEM stage smoke"
echo "core=$CORE_URL hub=$HUB_URL"
[ -n "$HUMAN_TOKEN" ]   || echo "  (no human token: authenticated checks will be skipped)"
[ -n "$SERVICE_TOKEN" ] || echo "  (no service token: machine API checks will be skipped)"
echo

echo "== Operational endpoints"
http_check "integration-core" "GET /health"  "$CORE_URL/health"  "200"
http_check "integration-core" "GET /readyz"  "$CORE_URL/readyz"  "200"
http_check "integration-core" "GET /metrics" "$CORE_URL/metrics" "200"
http_check "hub"              "GET /healthz" "$HUB_URL/healthz"  "200"

echo
echo "== Identity"
if [ -n "$AUTHENTIK_URL" ]; then
  http_check "authentik" "OIDC discovery" "$AUTHENTIK_URL/.well-known/openid-configuration" "200"
else
  record "authentik" "OIDC discovery" SKIP 0 "" "AUTHENTIK_URL is not configured"
fi

echo
echo "== Authorization boundary"
# These two are the security assertions of the smoke run, and they need no
# token at all: an unauthenticated caller must be rejected. A 200 here would be
# a finding worth stopping the acceptance for.
http_check "integration-core" "GET /api/v1/me rejects anonymous"        "$CORE_URL/api/v1/me"                          "401"
http_check "integration-core" "GET /api/v1/clients rejects anonymous"   "$CORE_URL/api/v1/clients"                     "401"
http_check "integration-core" "machine API rejects anonymous"           "$CORE_URL/api/service/v1/adapters/health"     "401"

echo
echo "== Normalized human API"
http_check "integration-core" "GET /api/v1/me"            "$CORE_URL/api/v1/me"            "200"     human
http_check "integration-core" "GET /api/v1/modules"       "$CORE_URL/api/v1/modules"       "200"     human
http_check "integration-core" "GET /api/v1/clients"       "$CORE_URL/api/v1/clients"       "200 403" human
http_check "integration-core" "GET /api/v1/projects"      "$CORE_URL/api/v1/projects"      "200 403" human
http_check "integration-core" "GET /api/v1/documents"     "$CORE_URL/api/v1/documents"     "200 403" human
http_check "integration-core" "GET /api/v1/notifications" "$CORE_URL/api/v1/notifications" "200"     human
http_check "integration-core" "GET /api/v1/search"        "$CORE_URL/api/v1/search?q=test" "200"     human

echo
echo "== Machine API"
http_check "integration-core" "GET /api/service/v1/whoami"          "$CORE_URL/api/service/v1/whoami"          "200" service
http_check "integration-core" "GET /api/service/v1/adapters"        "$CORE_URL/api/service/v1/adapters"        "200" service
http_check "integration-core" "GET /api/service/v1/adapters/health" "$CORE_URL/api/service/v1/adapters/health" "200" service

echo
echo "== Dependencies, through the platform's own readiness"
# PostgreSQL and NATS are on internal networks and are not probed directly:
# reaching them from outside would mean the topology is wrong. Their state is
# read from the platform that does depend on them.
readiness="$(curl --silent --max-time "$TIMEOUT" "$CORE_URL/readyz" 2>/dev/null)"
case "$readiness" in
  *'"database":"ok"'*|*'"database": "ok"'*)
    record "postgresql" "reachable through /readyz" PASS 0 "" "the Core reports the database as ok" ;;
  '')
    record "postgresql" "reachable through /readyz" FAIL 0 "" "no readiness response" ;;
  *)
    record "postgresql" "reachable through /readyz" FAIL 0 "" "the Core does not report the database as ok" ;;
esac
case "$readiness" in
  *'"nats":"ok"'*|*'"nats": "ok"'*)
    record "nats" "reachable through /readyz" PASS 0 "" "the Core reports NATS as ok" ;;
  *'degraded'*)
    # Degraded NATS is a documented, serving state. Reporting it as a failure
    # would make an intentional configuration look broken.
    record "nats" "reachable through /readyz" SKIP 0 "" "NATS is degraded or not configured; the platform serves without it and drops events" ;;
  '')
    record "nats" "reachable through /readyz" FAIL 0 "" "no readiness response" ;;
  *)
    record "nats" "reachable through /readyz" FAIL 0 "" "unrecognized readiness payload" ;;
esac

metrics="$(curl --silent --max-time "$TIMEOUT" "$CORE_URL/metrics" 2>/dev/null)"
case "$metrics" in
  *bsystem_build_info*) record "integration-core" "build metadata exposed" PASS 0 "" "bsystem_build_info is present" ;;
  '') record "integration-core" "build metadata exposed" FAIL 0 "" "no metrics response" ;;
  *)  record "integration-core" "build metadata exposed" FAIL 0 "" "bsystem_build_info is absent; the running commit cannot be identified" ;;
esac

echo
echo "== Owner-blocked"
record "platform" "customer isolation with real ownership" BLOCKED 0 "" "requires the authoritative customer ownership mapping; see TENANT-ISOLATION-MATRIX.md"
record "platform" "SLA timing" BLOCKED 0 "" "requires response and resolution targets per severity"

FINISHED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# --- report -----------------------------------------------------------------

commit_of() {
  local dir="$1"
  if [ -d "$dir/.git" ]; then
    git -C "$dir" rev-parse HEAD 2>/dev/null || echo "unknown"
  else
    echo "unavailable"
  fi
}

DEPLOY_COMMIT="$(commit_of .)"
CORE_COMMIT="$(commit_of ../bsystem-integration-core)"
HUB_COMMIT="$(commit_of ../bsystem-hub)"
DS_COMMIT="$(commit_of ../bsystem-design-system)"

{
  printf '{\n'
  printf '  "started_at": "%s",\n' "$STARTED_AT"
  printf '  "finished_at": "%s",\n' "$FINISHED_AT"
  printf '  "targets": {"core": "%s", "hub": "%s"},\n' "$(json_escape "$CORE_URL")" "$(json_escape "$HUB_URL")"
  printf '  "commits": {\n'
  printf '    "bsystem-deploy": "%s",\n' "$DEPLOY_COMMIT"
  printf '    "bsystem-integration-core": "%s",\n' "$CORE_COMMIT"
  printf '    "bsystem-hub": "%s",\n' "$HUB_COMMIT"
  printf '    "bsystem-design-system": "%s"\n' "$DS_COMMIT"
  printf '  },\n'
  printf '  "summary": {"pass": %d, "fail": %d, "skip": %d, "blocked": %d},\n' "$PASSED" "$FAILED" "$SKIPPED" "$BLOCKED"
  printf '  "checks": [\n'
  first=1
  while IFS=$'\t' read -r service check result duration request_id message; do
    [ "$first" -eq 1 ] || printf ',\n'
    first=0
    printf '    {"service": "%s", "check": "%s", "result": "%s", "duration_ms": %s, "request_id": "%s", "message": "%s"}' \
      "$(json_escape "$service")" "$(json_escape "$check")" "$result" "${duration:-0}" \
      "$(json_escape "$request_id")" "$(json_escape "$message")"
  done < "$RESULTS_FILE"
  printf '\n  ]\n}\n'
} > "$ARTIFACTS_DIR/stage-acceptance.json"

{
  printf '# Stage acceptance\n\n'
  printf '| | |\n| --- | --- |\n'
  printf '| Started | `%s` |\n' "$STARTED_AT"
  printf '| Finished | `%s` |\n' "$FINISHED_AT"
  printf '| Core | `%s` |\n' "$CORE_URL"
  printf '| HUB | `%s` |\n' "$HUB_URL"
  printf '\n## Commits\n\n| Repository | Commit |\n| --- | --- |\n'
  printf '| `bsystem-deploy` | `%s` |\n' "$DEPLOY_COMMIT"
  printf '| `bsystem-integration-core` | `%s` |\n' "$CORE_COMMIT"
  printf '| `bsystem-hub` | `%s` |\n' "$HUB_COMMIT"
  printf '| `bsystem-design-system` | `%s` |\n' "$DS_COMMIT"
  printf '\n## Result\n\n'
  printf '**%d passed, %d failed, %d skipped, %d blocked.**\n\n' "$PASSED" "$FAILED" "$SKIPPED" "$BLOCKED"
  printf '| Service | Check | Result | ms | Request ID | Note |\n'
  printf '| --- | --- | --- | --- | --- | --- |\n'
  while IFS=$'\t' read -r service check result duration request_id message; do
    printf '| %s | %s | %s | %s | `%s` | %s |\n' "$service" "$check" "$result" "${duration:-0}" "$request_id" "$message"
  done < "$RESULTS_FILE"
  printf '\nSKIP means a check could not run, usually for want of a token; it is not a pass.\n'
  printf 'BLOCKED means the check needs something only the owner can supply.\n'
  printf '\nNo credential, token or upstream payload appears in this report.\n'
} > "$ARTIFACTS_DIR/stage-acceptance.md"

echo
echo "report: $ARTIFACTS_DIR/stage-acceptance.json"
echo "report: $ARTIFACTS_DIR/stage-acceptance.md"
echo "$PASSED passed, $FAILED failed, $SKIPPED skipped, $BLOCKED blocked"

[ "$FAILED" -gt 0 ] && exit 1
exit 0
