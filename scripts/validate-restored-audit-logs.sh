#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: validate-restored-audit-logs.sh --restored-file <path> [options]

Options:
  --source-file <path>          Optional source sample file
  --restored-file <path>        Restored/replayed output file (required)
  --expected-sha256 <hex>       Expected sha256 (or EXPECTED_SHA256)
  --compare-source <0|1>        Compare source/restored hash if source exists (default: 1)
  --help                        Show this help

Env:
  SOURCE_FILE, RESTORED_FILE, EXPECTED_SHA256, COMPARE_SOURCE

Validation:
  1) If EXPECTED_SHA256 is set, restored hash must match it.
  2) If source file is provided and compare-source=1, source hash must equal restored hash.
EOF
}

SOURCE_FILE="${SOURCE_FILE:-}"
RESTORED_FILE="${RESTORED_FILE:-}"
EXPECTED_SHA256="${EXPECTED_SHA256:-}"
COMPARE_SOURCE="${COMPARE_SOURCE:-1}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --source-file) SOURCE_FILE="${2:-}"; shift 2 ;;
    --restored-file) RESTORED_FILE="${2:-}"; shift 2 ;;
    --expected-sha256) EXPECTED_SHA256="${2:-}"; shift 2 ;;
    --compare-source) COMPARE_SOURCE="${2:-}"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) echo "[validate-restored-audit-logs] Unknown arg: $1" >&2; usage; exit 1 ;;
  esac
done

[[ -n "$RESTORED_FILE" ]] || { echo "[validate-restored-audit-logs] Missing --restored-file or RESTORED_FILE" >&2; exit 1; }
[[ -f "$RESTORED_FILE" ]] || { echo "[validate-restored-audit-logs] Restored file not found: $RESTORED_FILE" >&2; exit 1; }

case "$COMPARE_SOURCE" in 0|1) ;; *) echo "[validate-restored-audit-logs] COMPARE_SOURCE must be 0 or 1" >&2; exit 1 ;; esac

hash_of() {
  python3 - "$1" <<'PY'
import hashlib
import sys
path = sys.argv[1]
h = hashlib.sha256()
with open(path, 'rb') as f:
    while True:
        chunk = f.read(1024 * 1024)
        if not chunk:
            break
        h.update(chunk)
print(h.hexdigest())
PY
}

restored_sha="$(hash_of "$RESTORED_FILE")"
echo "RESTORED_FILE=$RESTORED_FILE"
echo "RESTORED_SHA256=$restored_sha"

if [[ -n "$EXPECTED_SHA256" ]]; then
  if [[ "$restored_sha" != "$EXPECTED_SHA256" ]]; then
    echo "[validate-restored-audit-logs] SHA mismatch for restored file" >&2
    echo "[validate-restored-audit-logs] expected=$EXPECTED_SHA256" >&2
    echo "[validate-restored-audit-logs] actual=$restored_sha" >&2
    exit 1
  fi
  echo "[validate-restored-audit-logs] EXPECTED_SHA256 matched"
fi

if [[ -n "$SOURCE_FILE" && "$COMPARE_SOURCE" == "1" ]]; then
  [[ -f "$SOURCE_FILE" ]] || { echo "[validate-restored-audit-logs] Source file not found: $SOURCE_FILE" >&2; exit 1; }
  source_sha="$(hash_of "$SOURCE_FILE")"
  echo "SOURCE_FILE=$SOURCE_FILE"
  echo "SOURCE_SHA256=$source_sha"
  if [[ "$source_sha" != "$restored_sha" ]]; then
    echo "[validate-restored-audit-logs] Source/restored SHA mismatch" >&2
    echo "[validate-restored-audit-logs] source=$source_sha" >&2
    echo "[validate-restored-audit-logs] restored=$restored_sha" >&2
    exit 1
  fi
  echo "[validate-restored-audit-logs] Source/restored SHA matched"
fi

echo "[validate-restored-audit-logs] Validation passed"
