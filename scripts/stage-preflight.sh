#!/usr/bin/env bash
#
# Stage preflight: check that this environment could plausibly come up, before
# anything is started.
#
# It reads configuration and reports on it. It never prints a secret's value:
# secrets are reported as present, absent or still-a-placeholder, and nothing
# else. That rule is what makes it safe to run with output captured into a
# ticket, which is exactly what people do with preflight output.
#
# Exit codes:
#   0  no blocking failure
#   1  at least one blocking failure
#   2  the script could not run its checks at all
set -uo pipefail

ENV_FILE="${ENV_FILE:-.env}"
SKIP_NETWORK="${SKIP_NETWORK:-0}"
# SKIP_DOCKER exists so the script's own tests can exercise configuration
# checks on a machine with no Docker daemon. It is not for skipping an
# inconvenient failure during a real preflight.
SKIP_DOCKER="${SKIP_DOCKER:-0}"
FAIL=0
WARN=0

RED=''; YELLOW=''; GREEN=''; DIM=''; RESET=''
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
  RED=$'\033[31m'; YELLOW=$'\033[33m'; GREEN=$'\033[32m'; DIM=$'\033[2m'; RESET=$'\033[0m'
fi

pass() { printf '%s  PASS%s  %s\n' "$GREEN" "$RESET" "$1"; }
warn() { printf '%s  WARN%s  %s\n' "$YELLOW" "$RESET" "$1"; WARN=$((WARN + 1)); }
fail() { printf '%s  FAIL%s  %s\n' "$RED" "$RESET" "$1"; FAIL=$((FAIL + 1)); }
info() { printf '%s  ....%s  %s\n' "$DIM" "$RESET" "$1"; }
head_() { printf '\n== %s\n' "$1"; }

# --- configuration loading --------------------------------------------------

# load_env reads KEY=VALUE lines without executing the file. Sourcing a .env
# would run whatever is in it, which is an odd thing for a validator to do to a
# file it is about to call untrusted.
load_env() {
  local file="$1" line key value
  [ -f "$file" ] || return 0
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in
      ''|'#'*) continue ;;
    esac
    case "$line" in
      *=*) ;;
      *) continue ;;
    esac
    key="${line%%=*}"
    value="${line#*=}"
    key="${key#export }"
    key="$(printf '%s' "$key" | tr -d '[:space:]')"
    case "$key" in
      ''|*[!A-Za-z0-9_]*) continue ;;
    esac
    # Strip one layer of matching quotes, the way a .env reader would.
    case "$value" in
      \"*\") value="${value#\"}"; value="${value%\"}" ;;
      \'*\') value="${value#\'}"; value="${value%\'}" ;;
    esac
    if [ -z "${!key+x}" ]; then
      printf -v "$key" '%s' "$value"
      export "${key?}"
    fi
  done < "$file"
}

# is_placeholder reports whether a value is still a template rather than a
# setting. A placeholder that reaches a running service produces a failure far
# from its cause: "password authentication failed" tells you nothing about the
# CHANGE_ME two files away.
is_placeholder() {
  local value="$1"
  case "$value" in
    *CHANGE_ME*|*change_me*|*REPLACE_ME*|*TODO*|*xxxxx*|*XXXXX*) return 0 ;;
    # *://*.example and *://*.example/* would be redundant here: the two
    # patterns below already cover them.
    *.example|*.example/*|*.example.com*) return 0 ;;
    *stage.example*|*your-*) return 0 ;;
    *'<'*'>'*) return 0 ;;
  esac
  return 1
}

require_var() {
  local name="$1" kind="${2:-config}" value
  value="${!name:-}"
  if [ -z "$value" ]; then
    fail "$name is required and is not set"
    return
  fi
  if is_placeholder "$value"; then
    fail "$name still holds a placeholder value"
    return
  fi
  if [ "$kind" = "secret" ]; then
    pass "$name is set (${#value} characters)"
  else
    pass "$name is set: $value"
  fi
}

optional_var() {
  local name="$1" kind="${2:-config}" value
  value="${!name:-}"
  if [ -z "$value" ]; then
    info "$name is not set; the feature it enables stays off"
    return
  fi
  if is_placeholder "$value"; then
    fail "$name is set but still holds a placeholder value"
    return
  fi
  if [ "$kind" = "secret" ]; then
    pass "$name is set (${#value} characters)"
  else
    pass "$name is set: $value"
  fi
}

# --- URL, DNS and TCP -------------------------------------------------------

url_scheme() { printf '%s' "${1%%://*}"; }
url_host()   { local rest="${1#*://}"; rest="${rest%%/*}"; printf '%s' "${rest%%\?*}"; }
url_hostname() { local hostport; hostport="$(url_host "$1")"; hostport="${hostport##*@}"; printf '%s' "${hostport%%:*}"; }
url_port() {
  local hostport port
  hostport="$(url_host "$1")"; hostport="${hostport##*@}"
  case "$hostport" in
    *:*) port="${hostport##*:}" ;;
    *) case "$(url_scheme "$1")" in
         https) port=443 ;; http) port=80 ;; postgres|postgresql) port=5432 ;;
         nats) port=4222 ;; *) port='' ;;
       esac ;;
  esac
  printf '%s' "$port"
}

check_url() {
  local name="$1" require_tls="${2:-0}" value
  value="${!name:-}"
  [ -n "$value" ] || return 0
  case "$value" in
    *://*) ;;
    *) fail "$name is not a URL: it has no scheme"; return ;;
  esac
  local scheme host
  scheme="$(url_scheme "$value")"
  host="$(url_hostname "$value")"
  if [ -z "$host" ]; then
    fail "$name is not a URL: it has no host"
    return
  fi
  if [ "$require_tls" = "1" ] && [ "$scheme" != "https" ]; then
    case "$host" in
      localhost|127.0.0.1|::1) warn "$name is not HTTPS; acceptable only for a local run" ;;
      *) fail "$name must be HTTPS in stage (found $scheme)" ;;
    esac
  fi
  pass "$name parses as a URL (host $host)"

  [ "$SKIP_NETWORK" = "1" ] && return

  if command -v getent >/dev/null 2>&1; then
    if getent hosts "$host" >/dev/null 2>&1; then
      pass "$name resolves: $host"
    else
      case "$host" in
        postgres|nats|authentik-server|integration-core|hub)
          info "$name names a Compose service ($host); it resolves inside the stack, not here" ;;
        *) fail "$name does not resolve: $host" ;;
      esac
      return
    fi
  else
    info "getent is unavailable; skipping DNS for $host"
    return
  fi

  local port; port="$(url_port "$value")"
  [ -n "$port" ] || return
  # A read-only TCP connect. Nothing is sent, so this cannot alter anything on
  # the far side — which is the only kind of connectivity check a preflight has
  # any business making.
  if command -v nc >/dev/null 2>&1; then
    if nc -z -w 3 "$host" "$port" >/dev/null 2>&1; then
      pass "$name accepts TCP on $host:$port"
    else
      warn "$name does not accept TCP on $host:$port (a firewall, or not started yet)"
    fi
  elif timeout 3 bash -c "exec 3<>/dev/tcp/$host/$port" 2>/dev/null; then
    pass "$name accepts TCP on $host:$port"
  else
    warn "$name does not accept TCP on $host:$port (a firewall, or not started yet)"
  fi
}

# --- checks -----------------------------------------------------------------

main() {
  printf 'BSYSTEM stage preflight\n'
  printf '%sreading %s; secret values are never printed%s\n' "$DIM" "$ENV_FILE" "$RESET"

  head_ "Configuration file"
  if [ -f "$ENV_FILE" ]; then
    pass "$ENV_FILE exists"
    load_env "$ENV_FILE"
  else
    warn "$ENV_FILE not found; relying on the ambient environment"
  fi

  head_ "Required local files"
  for file in docker-compose.yml docker-compose.stage.yml .env.example; do
    if [ -f "$file" ]; then pass "$file exists"; else fail "$file is missing"; fi
  done

  head_ "Platform variables"
  require_var POSTGRES_PASSWORD secret
  require_var AUTHENTIK_SECRET_KEY secret
  require_var VITE_OIDC_AUTHORITY
  require_var VITE_OIDC_CLIENT_ID
  # The stage stack overrides every published port with STAGE_PUBLISH_ADDRESS.
  # BIND_ADDRESS governs the base stack and is read by nothing here, so
  # reporting on it would tell an operator their exposure is settled when the
  # variable that settles it has not been looked at.
  optional_var STAGE_PUBLISH_ADDRESS
  case "${STAGE_PUBLISH_ADDRESS:-}" in
    ''|127.0.0.1|localhost|::1|'[::1]') ;;
    0.0.0.0|'::'|'[::]'|'*')
      fail "STAGE_PUBLISH_ADDRESS is ${STAGE_PUBLISH_ADDRESS}: authentik, the Integration Core and the HUB are published on every interface" ;;
    *)
      warn "STAGE_PUBLISH_ADDRESS is ${STAGE_PUBLISH_ADDRESS}, not the loopback default; confirm that interface is behind the TLS proxy" ;;
  esac
  if [ -n "${BIND_ADDRESS:-}" ]; then
    warn "BIND_ADDRESS is set but the stage overlay does not read it; STAGE_PUBLISH_ADDRESS is what publishes the stage ports"
  fi

  if [ -n "${AUTHENTIK_SECRET_KEY:-}" ] && [ "${#AUTHENTIK_SECRET_KEY}" -lt 50 ]; then
    fail "AUTHENTIK_SECRET_KEY is shorter than the 50 characters authentik expects"
  fi

  head_ "Identity"
  check_url VITE_OIDC_AUTHORITY 1
  case "${VITE_OIDC_AUTHORITY:-}" in
    ''|*/) ;;
    *) fail "VITE_OIDC_AUTHORITY must end with a slash; an issuer mismatch rejects every token with no useful message" ;;
  esac
  optional_var AUTHENTIK_USERINFO_URL
  check_url AUTHENTIK_USERINFO_URL 0

  optional_var BSYSTEM_AUTHENTIK_ADMIN_TOKEN secret
  if [ -n "${BSYSTEM_AUTHENTIK_ADMIN_TOKEN:-}" ] && [ "${#BSYSTEM_AUTHENTIK_ADMIN_TOKEN}" -lt 32 ]; then
    fail "BSYSTEM_AUTHENTIK_ADMIN_TOKEN is shorter than 32 characters; generate a high-entropy token"
  fi

  # Local token validation. Empty keeps the default, where the Core asks
  # UserInfo once per request.
  optional_var OIDC_ISSUER_URL
  check_url OIDC_ISSUER_URL 1
  if [ -n "${OIDC_ISSUER_URL:-}" ] && [ -z "${OIDC_AUDIENCE:-}" ]; then
    # Not fatal, and not silent. With no audience configured the platform
    # accepts a token minted for any client of this issuer — which is fine on
    # an authentik with one application and an authorization hole on one with
    # several, and the preflight cannot tell which this is.
    warn "OIDC_ISSUER_URL is set and OIDC_AUDIENCE is not: a token minted for any client of this issuer will be accepted"
  fi
  # Bootstrap credentials outlive their purpose silently.
  #
  # authentik reads them on every start, not only the first, so a password
  # left in .env after the account exists is a static administrator credential
  # in a running container's environment — and nothing else would ever say so.
  if [ -n "${AUTHENTIK_BOOTSTRAP_PASSWORD:-}" ] || [ -n "${AUTHENTIK_BOOTSTRAP_TOKEN:-}" ]; then
    warn "AUTHENTIK_BOOTSTRAP_PASSWORD/TOKEN are set: clear them once akadmin exists, or they stay a static administrator credential in the container's environment"
  fi

  # There is deliberately no check that one of the two is set. The base
  # Compose file supplies AUTHENTIK_USERINFO_URL itself, pointing at the
  # authentik in the same stack, so an empty value here does not mean the
  # platform cannot authenticate. A check that read it as one was written,
  # failed two existing tests, and was wrong rather than the tests.

  head_ "Source systems"
  for pair in "ESPOCRM_URL ESPOCRM_API_KEY" "REDMINE_URL REDMINE_API_KEY" "OUTLINE_URL OUTLINE_API_KEY"; do
    set -- $pair
    local url_name="$1" key_name="$2"
    optional_var "$url_name"
    check_url "$url_name" 1
    if [ -n "${!url_name:-}" ]; then
      optional_var "$key_name" secret
      if [ "$url_name" = "OUTLINE_URL" ] && [ -z "${OUTLINE_API_KEY:-}" ]; then
        fail "OUTLINE_API_KEY is required when OUTLINE_URL is set: Outline has no anonymous read surface"
      fi
    fi
  done

  head_ "Optional subsystems"
  optional_var OPENSEARCH_URL
  check_url OPENSEARCH_URL 0
  optional_var AI_PROVIDER
  if [ "${AI_PROVIDER:-}" = "openai" ]; then
    require_var OPENAI_API_KEY secret
  fi

  head_ "Container runtime"
  if [ "$SKIP_DOCKER" = "1" ]; then
    info "SKIP_DOCKER is set; the container runtime was not checked"
  elif command -v docker >/dev/null 2>&1; then
    pass "docker is on PATH"
    if docker info >/dev/null 2>&1; then
      pass "the Docker daemon is reachable"
    else
      fail "docker is installed but the daemon is not reachable"
    fi
    if docker compose version >/dev/null 2>&1; then
      pass "docker compose is available"
      if docker compose -f docker-compose.yml -f docker-compose.stage.yml config --quiet >/dev/null 2>&1; then
        pass "the stage Compose configuration renders"
      else
        fail "the stage Compose configuration does not render; run the same command to see why"
      fi
    else
      fail "docker compose is not available"
    fi
  else
    fail "docker is not on PATH"
  fi

  head_ "Result"
  printf '%d failure(s), %d warning(s)\n' "$FAIL" "$WARN"
  if [ "$FAIL" -gt 0 ]; then
    printf '%spreflight blocked: fix the failures above before starting the stack%s\n' "$RED" "$RESET"
    return 1
  fi
  printf '%spreflight passed%s\n' "$GREEN" "$RESET"
  return 0
}

main "$@"
