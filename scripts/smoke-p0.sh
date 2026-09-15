#!/bin/sh
set -eu

HUB_URL="${HUB_URL:-http://localhost:8081}"
CORE_URL="${CORE_URL:-http://localhost:8080}"
AUTHENTIK_URL="${AUTHENTIK_URL:-http://localhost:9000}"

check_200() {
  name="$1"
  url="$2"
  code="$(curl -fsS -o /tmp/bsystem-smoke-body -w '%{http_code}' "$url" || true)"
  if [ "$code" != "200" ]; then
    echo "FAIL: $name expected 200, got $code"
    cat /tmp/bsystem-smoke-body 2>/dev/null || true
    exit 1
  fi
  echo "OK: $name"
}

check_200 "Integration Core health" "$CORE_URL/health"
check_200 "HUB health" "$HUB_URL/healthz"
check_200 "authentik HTTP" "$AUTHENTIK_URL/-/health/live/"

code="$(curl -sS -o /tmp/bsystem-smoke-body -w '%{http_code}' "$CORE_URL/api/v1/me" || true)"
if [ "$code" != "401" ]; then
  echo "FAIL: unauthenticated /api/v1/me expected 401, got $code"
  cat /tmp/bsystem-smoke-body 2>/dev/null || true
  exit 1
fi
echo "OK: unauthenticated /api/v1/me is protected"

echo "BSYSTEM P0 smoke checks passed."
