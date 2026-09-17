#!/usr/bin/env bash
#
# Generate a release manifest: exactly what this checkout would deploy.
#
# It answers the question that gets asked after something goes wrong and nobody
# can remember what was running. Everything here is public identity — commits,
# tags, hashes. Nothing is read from a .env and no credential is touched.
#
# Writes JSON to stdout.
set -uo pipefail

CORE_DIR="${CORE_DIR:-../bsystem-integration-core}"
HUB_DIR="${HUB_DIR:-../bsystem-hub}"
DS_DIR="${DS_DIR:-../bsystem-design-system}"

json_string() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

repo_field() {
  local dir="$1" what="$2"
  if [ ! -d "$dir/.git" ]; then printf 'unavailable'; return; fi
  case "$what" in
    commit) git -C "$dir" rev-parse HEAD 2>/dev/null || printf 'unknown' ;;
    branch) git -C "$dir" rev-parse --abbrev-ref HEAD 2>/dev/null || printf 'unknown' ;;
    dirty)  if [ -n "$(git -C "$dir" status --porcelain 2>/dev/null)" ]; then printf 'true'; else printf 'false'; fi ;;
    date)   git -C "$dir" log -1 --format=%cI 2>/dev/null || printf 'unknown' ;;
  esac
}

repo_object() {
  local name="$1" dir="$2" dirty
  # "dirty" is a JSON boolean, so an unavailable repository cannot borrow the
  # "unavailable" string the other fields use: it would emit a bare token and
  # the document would not parse. This only shows up where a sibling is not
  # checked out — which is every CI run, and was not reproduced locally until
  # a test pinned it.
  if [ -d "$dir/.git" ]; then
    dirty="$(repo_field "$dir" dirty)"
  else
    dirty="null"
  fi
  printf '    "%s": {"commit": "%s", "branch": "%s", "committed_at": "%s", "dirty": %s}' \
    "$name" "$(repo_field "$dir" commit)" "$(json_string "$(repo_field "$dir" branch)")" \
    "$(repo_field "$dir" date)" "$dirty"
}

# The schema level this build would apply — read from the migration files the
# Core carries, which is what a deployment of this checkout would run. The
# level a database is actually at is a different number, reported by
# bsystem_schema_migrations_applied on /metrics; comparing the two is how you
# find a database that is behind its code.
schema_level() {
  local dir="$CORE_DIR/internal/platformdb/migrations"
  [ -d "$dir" ] || { printf 'unavailable'; return; }
  local latest
  latest="$(ls -1 "$dir"/*.sql 2>/dev/null | sed 's#.*/##' | sort | tail -1)"
  printf '%s' "${latest:-none}"
}

schema_count() {
  local dir="$CORE_DIR/internal/platformdb/migrations"
  [ -d "$dir" ] || { printf '0'; return; }
  ls -1 "$dir"/*.sql 2>/dev/null | wc -l | tr -d ' '
}

# A hash rather than a version: the OpenAPI document has no version of its own
# that moves per change, and a hash is what tells you whether the contract a
# client generated from is the contract being served.
openapi_hash() {
  local file="$CORE_DIR/docs/openapi.yaml"
  [ -f "$file" ] || { printf 'unavailable'; return; }
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | cut -c1-16
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | cut -c1-16
  else
    printf 'unavailable'
  fi
}

openapi_version() {
  local file="$CORE_DIR/docs/openapi.yaml"
  [ -f "$file" ] || { printf 'unavailable'; return; }
  grep -m1 -E '^\s+version:' "$file" | sed 's/.*version:[[:space:]]*//; s/["'"'"']//g' | tr -d '\r' || printf 'unknown'
}

design_system_version() {
  local file="$DS_DIR/package.json"
  [ -f "$file" ] || { printf 'unavailable'; return; }
  grep -m1 '"version"' "$file" | sed 's/.*"version"[[:space:]]*:[[:space:]]*"//; s/".*//' || printf 'unknown'
}

# The images this release is made of, as recorded at the moment they were
# built. Read from a file rather than inspected here, so the manifest describes
# the artifact that was tested and scanned rather than whatever happens to be
# tagged when the manifest is generated — which is the same thing only if
# nothing rebuilt in between, and "only if" is what the pipeline is for. See
# scripts/image-digests.py.
images_object() {
  local file="${RELEASE_IMAGES:-}"
  if [ -z "$file" ] || [ ! -f "$file" ]; then
    printf 'null'
    return
  fi
  # Emitted verbatim: it is already JSON, written by the recorder.
  cat "$file"
}

cat <<JSON
{
  "generated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "images": $(images_object),
  "repositories": {
$(repo_object "bsystem-deploy" "."),
$(repo_object "bsystem-integration-core" "$CORE_DIR"),
$(repo_object "bsystem-hub" "$HUB_DIR"),
$(repo_object "bsystem-design-system" "$DS_DIR")
  },
  "schema": {
    "level": "$(schema_level)",
    "migrations": $(schema_count)
  },
  "openapi": {
    "version": "$(openapi_version)",
    "sha256_prefix": "$(openapi_hash)"
  },
  "design_system": {
    "version": "$(design_system_version)"
  }
}
JSON
