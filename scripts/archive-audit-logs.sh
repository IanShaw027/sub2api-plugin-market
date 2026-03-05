#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TABLE_NAME="plugin_market_audit_events"
TIME_COLUMN="created_at"

PG_DSN="${PG_DSN:-}"
ARCHIVE_BEFORE_DAYS="${ARCHIVE_BEFORE_DAYS:-90}"
ARCHIVE_MODE="${ARCHIVE_MODE:-move}"
EXPORT_FORMAT="${EXPORT_FORMAT:-csv}"
COLD_STORAGE_MODE="${COLD_STORAGE_MODE:-local}"
S3_URI="${S3_URI:-}"
DRY_RUN="${DRY_RUN:-1}"
OUTPUT_DIR="${OUTPUT_DIR:-${SCRIPT_DIR}/../data/audit-archive}"

usage() {
  cat <<'HELP'
Usage: archive-audit-logs.sh [options]

Options:
  --pg-dsn <dsn>                 PostgreSQL DSN (required when DRY_RUN=0)
  --before-days <days>           Archive threshold in days (default: 90)
  --archive-mode <move|copy|purge>
                                 Archive mode (default: move)
  --format <csv|jsonl>           Export format (default: csv)
  --cold-storage <local|local_and_s3|none>
                                 Cold storage mode (default: local)
  --s3-uri <s3://bucket/path>    S3 target URI (required when local_and_s3)
  --dry-run <0|1>                Dry run flag (default: 1)
  --output-dir <path>            Output directory (default: scripts/../data/audit-archive)
  --help                         Show this help

Env equivalents:
  PG_DSN, ARCHIVE_BEFORE_DAYS, ARCHIVE_MODE,
  EXPORT_FORMAT, COLD_STORAGE_MODE, S3_URI,
  DRY_RUN, OUTPUT_DIR
HELP
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --pg-dsn) PG_DSN="${2:-}"; shift 2 ;;
    --before-days) ARCHIVE_BEFORE_DAYS="${2:-}"; shift 2 ;;
    --archive-mode) ARCHIVE_MODE="${2:-}"; shift 2 ;;
    --format) EXPORT_FORMAT="${2:-}"; shift 2 ;;
    --cold-storage) COLD_STORAGE_MODE="${2:-}"; shift 2 ;;
    --s3-uri) S3_URI="${2:-}"; shift 2 ;;
    --dry-run) DRY_RUN="${2:-}"; shift 2 ;;
    --output-dir) OUTPUT_DIR="${2:-}"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *)
      echo "[archive-audit-logs] Unknown arg: $1" >&2
      usage
      exit 1
      ;;
  esac
done

case "$ARCHIVE_MODE" in
  move|copy|purge) ;;
  *) echo "[archive-audit-logs] ARCHIVE_MODE must be move|copy|purge" >&2; exit 1 ;;
esac

case "$EXPORT_FORMAT" in
  csv|jsonl) ;;
  *) echo "[archive-audit-logs] EXPORT_FORMAT must be csv|jsonl" >&2; exit 1 ;;
esac

case "$COLD_STORAGE_MODE" in
  local|local_and_s3|none) ;;
  *) echo "[archive-audit-logs] COLD_STORAGE_MODE must be local|local_and_s3|none" >&2; exit 1 ;;
esac

case "$DRY_RUN" in
  0|1) ;;
  *) echo "[archive-audit-logs] DRY_RUN must be 0 or 1" >&2; exit 1 ;;
esac

if ! [[ "$ARCHIVE_BEFORE_DAYS" =~ ^[0-9]+$ ]]; then
  echo "[archive-audit-logs] ARCHIVE_BEFORE_DAYS must be a non-negative integer" >&2
  exit 1
fi

if [[ "$COLD_STORAGE_MODE" == "local_and_s3" && -z "$S3_URI" ]]; then
  echo "[archive-audit-logs] S3_URI is required when COLD_STORAGE_MODE=local_and_s3" >&2
  exit 1
fi

if [[ "$ARCHIVE_MODE" != "purge" && "$COLD_STORAGE_MODE" == "none" ]]; then
  echo "[archive-audit-logs] COLD_STORAGE_MODE=none is invalid when ARCHIVE_MODE=${ARCHIVE_MODE}" >&2
  exit 1
fi

if [[ "$DRY_RUN" == "0" && -z "$PG_DSN" ]]; then
  echo "[archive-audit-logs] PG_DSN is required when DRY_RUN=0" >&2
  exit 1
fi

have_psql=0
if command -v psql >/dev/null 2>&1; then
  have_psql=1
fi

if [[ "$DRY_RUN" == "0" && "$have_psql" -ne 1 ]]; then
  echo "[archive-audit-logs] psql not found in PATH; cannot run with DRY_RUN=0" >&2
  exit 1
fi

sql_escape_literal() {
  local value="$1"
  printf "%s" "${value//\'/\'\'}"
}

WINDOW_EXPR="NOW() - INTERVAL '${ARCHIVE_BEFORE_DAYS} days'"
COUNT_SQL="SELECT COUNT(*) FROM ${TABLE_NAME} WHERE ${TIME_COLUMN} < ${WINDOW_EXPR};"
DELETE_SQL="DELETE FROM ${TABLE_NAME} WHERE ${TIME_COLUMN} < ${WINDOW_EXPR};"

TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
EXT="csv"
if [[ "$EXPORT_FORMAT" == "jsonl" ]]; then
  EXT="jsonl"
fi
EXPORT_FILE="${OUTPUT_DIR}/${TABLE_NAME}-before-${ARCHIVE_BEFORE_DAYS}d-${TIMESTAMP}.${EXT}"

CANDIDATE_ROWS="N/A"
if [[ -n "$PG_DSN" && "$have_psql" -eq 1 ]]; then
  CANDIDATE_ROWS="$(psql "$PG_DSN" -v ON_ERROR_STOP=1 -At -c "$COUNT_SQL")"
elif [[ "$DRY_RUN" == "0" ]]; then
  echo "[archive-audit-logs] Unexpected state: missing DB count capability" >&2
  exit 1
fi

echo "[archive-audit-logs] table=${TABLE_NAME}"
echo "[archive-audit-logs] threshold_days=${ARCHIVE_BEFORE_DAYS}"
echo "[archive-audit-logs] window=${TIME_COLUMN} < ${WINDOW_EXPR}"
echo "[archive-audit-logs] archive_mode=${ARCHIVE_MODE}"
echo "[archive-audit-logs] export_format=${EXPORT_FORMAT}"
echo "[archive-audit-logs] cold_storage_mode=${COLD_STORAGE_MODE}"
echo "[archive-audit-logs] dry_run=${DRY_RUN}"
echo "[archive-audit-logs] output_dir=${OUTPUT_DIR}"
echo "[archive-audit-logs] candidate_rows=${CANDIDATE_ROWS}"
echo "[archive-audit-logs] planned_export_file=${EXPORT_FILE}"

if [[ "$DRY_RUN" == "1" ]]; then
  echo "[archive-audit-logs] dry-run: no DB write/export/upload executed"
  if [[ "$ARCHIVE_MODE" == "copy" || "$ARCHIVE_MODE" == "move" ]]; then
    echo "[archive-audit-logs] plan: export rows older than ${ARCHIVE_BEFORE_DAYS} days"
    if [[ "$COLD_STORAGE_MODE" == "local_and_s3" ]]; then
      echo "[archive-audit-logs] plan: upload to ${S3_URI%/}/$(basename "$EXPORT_FILE")"
    fi
  fi
  if [[ "$ARCHIVE_MODE" == "move" || "$ARCHIVE_MODE" == "purge" ]]; then
    echo "[archive-audit-logs] plan: delete rows older than ${ARCHIVE_BEFORE_DAYS} days"
  fi
  exit 0
fi

if [[ "$ARCHIVE_MODE" == "copy" || "$ARCHIVE_MODE" == "move" ]]; then
  mkdir -p "$OUTPUT_DIR"

  SELECT_SQL="SELECT * FROM ${TABLE_NAME} WHERE ${TIME_COLUMN} < ${WINDOW_EXPR} ORDER BY ${TIME_COLUMN} ASC"
  ESCAPED_EXPORT_FILE="$(sql_escape_literal "$EXPORT_FILE")"

  if [[ "$EXPORT_FORMAT" == "csv" ]]; then
    psql "$PG_DSN" -v ON_ERROR_STOP=1 -c "\\copy (${SELECT_SQL}) TO '${ESCAPED_EXPORT_FILE}' WITH (FORMAT csv, HEADER true)"
  else
    psql "$PG_DSN" -v ON_ERROR_STOP=1 -c "\\copy (SELECT row_to_json(t)::text FROM (${SELECT_SQL}) AS t) TO '${ESCAPED_EXPORT_FILE}'"
  fi

  echo "[archive-audit-logs] exported_file=${EXPORT_FILE}"

  if [[ "$COLD_STORAGE_MODE" == "local_and_s3" ]]; then
    if ! command -v aws >/dev/null 2>&1; then
      echo "[archive-audit-logs] aws CLI is required when COLD_STORAGE_MODE=local_and_s3" >&2
      exit 1
    fi

    S3_TARGET="${S3_URI%/}/$(basename "$EXPORT_FILE")"
    aws s3 cp "$EXPORT_FILE" "$S3_TARGET"
    echo "[archive-audit-logs] uploaded_s3=${S3_TARGET}"
  fi
fi

if [[ "$ARCHIVE_MODE" == "move" || "$ARCHIVE_MODE" == "purge" ]]; then
  DELETE_COUNT="$(psql "$PG_DSN" -v ON_ERROR_STOP=1 -At -c "WITH deleted AS (${DELETE_SQL} RETURNING 1) SELECT COUNT(*) FROM deleted;")"
  echo "[archive-audit-logs] deleted_rows=${DELETE_COUNT}"
fi

echo "[archive-audit-logs] done"
