#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: query-restored-audit-logs.sh --input <path> [options]

Options:
  --input <path>             Input restored JSONL file (or INPUT_FILE) (required)
  --page <n>                 Page number, starts from 1 (default: 1)
  --page-size <n>            Page size, must be > 0 (default: 20)
  --from-time <iso8601>      Inclusive lower bound for event time
  --to-time <iso8601>        Inclusive upper bound for event time
  --action <value>           Exact match on action field
  --actor <value>            Exact match on actor field
  --contains <text>          Substring match against raw JSON line text
  --help                     Show this help

Env equivalents:
  INPUT_FILE, PAGE, PAGE_SIZE, FROM_TIME, TO_TIME, ACTION, ACTOR, CONTAINS

Output:
  - stdout: paged JSON array
  - stderr summary keys: TOTAL_MATCHED, TOTAL_PAGES, CURRENT_PAGE, PAGE_SIZE, INVALID_LINES
USAGE
}

INPUT_FILE="${INPUT_FILE:-}"
PAGE="${PAGE:-1}"
PAGE_SIZE="${PAGE_SIZE:-20}"
FROM_TIME="${FROM_TIME:-}"
TO_TIME="${TO_TIME:-}"
ACTION="${ACTION:-}"
ACTOR="${ACTOR:-}"
CONTAINS="${CONTAINS:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --input) INPUT_FILE="${2:-}"; shift 2 ;;
    --page) PAGE="${2:-}"; shift 2 ;;
    --page-size) PAGE_SIZE="${2:-}"; shift 2 ;;
    --from-time) FROM_TIME="${2:-}"; shift 2 ;;
    --to-time) TO_TIME="${2:-}"; shift 2 ;;
    --action) ACTION="${2:-}"; shift 2 ;;
    --actor) ACTOR="${2:-}"; shift 2 ;;
    --contains) CONTAINS="${2:-}"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) echo "[query-restored-audit-logs] Unknown arg: $1" >&2; usage; exit 1 ;;
  esac
done

[[ -n "$INPUT_FILE" ]] || { echo "[query-restored-audit-logs] Missing --input or INPUT_FILE" >&2; exit 1; }
[[ -f "$INPUT_FILE" ]] || { echo "[query-restored-audit-logs] Input not found: $INPUT_FILE" >&2; exit 1; }

case "$PAGE" in ''|*[!0-9]*) echo "[query-restored-audit-logs] PAGE must be a positive integer" >&2; exit 1 ;; esac
case "$PAGE_SIZE" in ''|*[!0-9]*) echo "[query-restored-audit-logs] PAGE_SIZE must be a positive integer" >&2; exit 1 ;; esac
[[ "$PAGE" -ge 1 ]] || { echo "[query-restored-audit-logs] PAGE must be >= 1" >&2; exit 1; }
[[ "$PAGE_SIZE" -ge 1 ]] || { echo "[query-restored-audit-logs] PAGE_SIZE must be >= 1" >&2; exit 1; }

python3 - "$INPUT_FILE" "$PAGE" "$PAGE_SIZE" "$FROM_TIME" "$TO_TIME" "$ACTION" "$ACTOR" "$CONTAINS" <<'PY'
import datetime as dt
import json
import sys

input_file, page_s, page_size_s, from_time_s, to_time_s, action, actor, contains = sys.argv[1:]

page = int(page_s)
page_size = int(page_size_s)


def parse_iso8601(value):
    if not value:
        return None
    s = value.strip()
    if not s:
        return None
    if s.endswith('Z'):
        s = s[:-1] + '+00:00'
    try:
        parsed = dt.datetime.fromisoformat(s)
    except ValueError:
        raise SystemExit(f"[query-restored-audit-logs] Invalid ISO8601 time: {value}")
    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=dt.timezone.utc)
    return parsed


def extract_event_time(record):
    for key in ("event_time", "timestamp", "created_at", "time"):
        value = record.get(key)
        if not isinstance(value, str):
            continue
        s = value.strip()
        if not s:
            continue
        if s.endswith('Z'):
            s = s[:-1] + '+00:00'
        try:
            parsed = dt.datetime.fromisoformat(s)
        except ValueError:
            continue
        if parsed.tzinfo is None:
            parsed = parsed.replace(tzinfo=dt.timezone.utc)
        return parsed
    return None


from_time = parse_iso8601(from_time_s)
to_time = parse_iso8601(to_time_s)

if from_time and to_time and from_time > to_time:
    raise SystemExit("[query-restored-audit-logs] --from-time must be <= --to-time")

matched_records = []
invalid_lines = 0

with open(input_file, 'r', encoding='utf-8') as fp:
    for raw_line in fp:
        line = raw_line.strip()
        if not line:
            continue

        if contains and contains not in line:
            continue

        try:
            record = json.loads(line)
        except json.JSONDecodeError:
            invalid_lines += 1
            continue

        if not isinstance(record, dict):
            # Non-object JSON is considered valid JSONL entry; keep for contains-only use.
            # But action/actor/time filters require object fields and thus naturally fail below.
            record_obj = {}
        else:
            record_obj = record

        if action and record_obj.get("action") != action:
            continue
        if actor and record_obj.get("actor") != actor:
            continue

        event_time = extract_event_time(record_obj)
        if from_time and (event_time is None or event_time < from_time):
            continue
        if to_time and (event_time is None or event_time > to_time):
            continue

        matched_records.append(record)

total_matched = len(matched_records)
total_pages = (total_matched + page_size - 1) // page_size if total_matched > 0 else 0
start = (page - 1) * page_size
end = start + page_size
page_items = matched_records[start:end] if start < total_matched else []

json.dump(page_items, sys.stdout, ensure_ascii=False, indent=2)
sys.stdout.write("\n")

sys.stderr.write(f"TOTAL_MATCHED={total_matched}\n")
sys.stderr.write(f"TOTAL_PAGES={total_pages}\n")
sys.stderr.write(f"CURRENT_PAGE={page}\n")
sys.stderr.write(f"PAGE_SIZE={page_size}\n")
sys.stderr.write(f"INVALID_LINES={invalid_lines}\n")
PY
