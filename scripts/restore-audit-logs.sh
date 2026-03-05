#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: restore-audit-logs.sh --input <path> [options]

Options:
  --input <path>                 Input file (CSV/JSONL)
  --format <csv|jsonl|auto>      Input format (default: auto)
  --output <path>                Output JSONL path (default: temp file)
  --dry-run <0|1>                Dry run flag (default: 1)
  --window-start <iso8601>       Inclusive start time filter
  --window-end <iso8601>         Inclusive end time filter
  --dedup-mode <none|target|batch|both>
                                 Dedup mode (default: both)
  --key-field <name>             Dedup key field (default: id)
  --target-keys-file <path>      Existing keys from target table (one key per line)
  --help                         Show this help

Env equivalents:
  INPUT_FILE, INPUT_FORMAT, OUTPUT_FILE, DRY_RUN,
  WINDOW_START, WINDOW_END, DEDUP_MODE, KEY_FIELD, TARGET_KEYS_FILE

Summary output keys:
  TOTAL_INPUT, WINDOW_FILTERED, DUPLICATE_TARGET,
  DUPLICATE_BATCH, TO_RESTORE, OUTPUT_FILE
EOF
}

INPUT_FILE="${INPUT_FILE:-}"
INPUT_FORMAT="${INPUT_FORMAT:-auto}"
OUTPUT_FILE="${OUTPUT_FILE:-}"
DRY_RUN="${DRY_RUN:-1}"
WINDOW_START="${WINDOW_START:-}"
WINDOW_END="${WINDOW_END:-}"
DEDUP_MODE="${DEDUP_MODE:-both}"
KEY_FIELD="${KEY_FIELD:-id}"
TARGET_KEYS_FILE="${TARGET_KEYS_FILE:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --input) INPUT_FILE="${2:-}"; shift 2 ;;
    --format) INPUT_FORMAT="${2:-}"; shift 2 ;;
    --output) OUTPUT_FILE="${2:-}"; shift 2 ;;
    --dry-run) DRY_RUN="${2:-}"; shift 2 ;;
    --window-start) WINDOW_START="${2:-}"; shift 2 ;;
    --window-end) WINDOW_END="${2:-}"; shift 2 ;;
    --dedup-mode) DEDUP_MODE="${2:-}"; shift 2 ;;
    --key-field) KEY_FIELD="${2:-}"; shift 2 ;;
    --target-keys-file) TARGET_KEYS_FILE="${2:-}"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) echo "[restore-audit-logs] Unknown arg: $1" >&2; usage; exit 1 ;;
  esac
done

[[ -n "$INPUT_FILE" ]] || { echo "[restore-audit-logs] Missing --input or INPUT_FILE" >&2; exit 1; }
[[ -f "$INPUT_FILE" ]] || { echo "[restore-audit-logs] Input not found: $INPUT_FILE" >&2; exit 1; }

case "$INPUT_FORMAT" in auto|csv|jsonl) ;; *) echo "[restore-audit-logs] INPUT_FORMAT must be auto|csv|jsonl" >&2; exit 1 ;; esac
case "$DRY_RUN" in 0|1) ;; *) echo "[restore-audit-logs] DRY_RUN must be 0 or 1" >&2; exit 1 ;; esac
case "$DEDUP_MODE" in none|target|batch|both) ;; *) echo "[restore-audit-logs] DEDUP_MODE must be none|target|batch|both" >&2; exit 1 ;; esac

if [[ -z "$OUTPUT_FILE" ]]; then
  OUTPUT_FILE="$(mktemp /tmp/restored-audit-logs.XXXXXX.jsonl)"
fi

python3 - "$INPUT_FILE" "$INPUT_FORMAT" "$OUTPUT_FILE" "$WINDOW_START" "$WINDOW_END" "$DEDUP_MODE" "$KEY_FIELD" "$TARGET_KEYS_FILE" <<'PY'
import csv
import datetime as dt
import json
import sys
from pathlib import Path

input_file, input_format, output_file, window_start, window_end, dedup_mode, key_field, target_keys_file = sys.argv[1:]

def parse_time(value):
    if not value:
        return None
    s = value.strip()
    if s.endswith('Z'):
        s = s[:-1] + '+00:00'
    try:
        return dt.datetime.fromisoformat(s)
    except ValueError:
        return None

def normalize_time(record):
    for key in ("event_time", "timestamp", "created_at", "time"):
        value = record.get(key)
        if isinstance(value, str):
            t = parse_time(value)
            if t is not None:
                return t
    return None

def detect_format(path):
    p = path.lower()
    if p.endswith('.csv'):
        return 'csv'
    if p.endswith('.jsonl') or p.endswith('.ndjson'):
        return 'jsonl'
    return 'jsonl'

fmt = detect_format(input_file) if input_format == 'auto' else input_format
start = parse_time(window_start)
end = parse_time(window_end)

target_keys = set()
if dedup_mode in ('target', 'both') and target_keys_file:
    f = Path(target_keys_file)
    if f.exists():
        target_keys = {line.strip() for line in f.read_text().splitlines() if line.strip()}

counts = {
    'total': 0,
    'window_filtered': 0,
    'dup_target': 0,
    'dup_batch': 0,
    'selected': 0,
}
batch_seen = set()

def in_window(record):
    t = normalize_time(record)
    if start and (t is None or t < start):
        return False
    if end and (t is None or t > end):
        return False
    return True

with open(output_file, 'w', encoding='utf-8') as out:
    if fmt == 'csv':
        with open(input_file, 'r', encoding='utf-8', newline='') as fp:
            rows = csv.DictReader(fp)
            for row in rows:
                counts['total'] += 1
                if not in_window(row):
                    counts['window_filtered'] += 1
                    continue
                dedup_key = str(row.get(key_field, '')).strip()
                if dedup_mode in ('target', 'both') and dedup_key and dedup_key in target_keys:
                    counts['dup_target'] += 1
                    continue
                if dedup_mode in ('batch', 'both') and dedup_key and dedup_key in batch_seen:
                    counts['dup_batch'] += 1
                    continue
                if dedup_key:
                    batch_seen.add(dedup_key)
                out.write(json.dumps(row, ensure_ascii=False) + '\n')
                counts['selected'] += 1
    else:
        with open(input_file, 'r', encoding='utf-8') as fp:
            for line in fp:
                line = line.strip()
                if not line:
                    continue
                row = json.loads(line)
                counts['total'] += 1
                if not in_window(row):
                    counts['window_filtered'] += 1
                    continue
                dedup_key = str(row.get(key_field, '')).strip()
                if dedup_mode in ('target', 'both') and dedup_key and dedup_key in target_keys:
                    counts['dup_target'] += 1
                    continue
                if dedup_mode in ('batch', 'both') and dedup_key and dedup_key in batch_seen:
                    counts['dup_batch'] += 1
                    continue
                if dedup_key:
                    batch_seen.add(dedup_key)
                out.write(json.dumps(row, ensure_ascii=False) + '\n')
                counts['selected'] += 1

print(f"TOTAL_INPUT={counts['total']}")
print(f"WINDOW_FILTERED={counts['window_filtered']}")
print(f"DUPLICATE_TARGET={counts['dup_target']}")
print(f"DUPLICATE_BATCH={counts['dup_batch']}")
print(f"TO_RESTORE={counts['selected']}")
print(f"OUTPUT_FILE={output_file}")
PY

if [[ "$DRY_RUN" == "1" ]]; then
  echo "[restore-audit-logs] Dry-run complete"
  exit 0
fi

echo "[restore-audit-logs] DRY_RUN=0 is intentionally disabled in this minimal script" >&2
echo "[restore-audit-logs] Use generated OUTPUT_FILE for controlled replay into DB" >&2
exit 1
