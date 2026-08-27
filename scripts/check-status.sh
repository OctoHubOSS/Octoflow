#!/usr/bin/env bash
# This deliberately never touches Postgres directly /api/health itself
# always returns a JSON body even when its own DB query fails (see
# webserver/ontos/status.go's ApiHealth), so a plain HTTPS GET is enough to
# observe "webserver down" (connection failure) vs "webserver up, DB down"
# (valid JSON with database:false) vs fully healthy.
set -euo pipefail

API_URL="${OCTOFLOW_API_URL:-https://v2.gitlogs.xyz}"
HISTORY_FILE="${1:-status-history.ndjson}"
MAX_LINES=30000
TIMEOUT_SECS=10

checked_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
start_ms=$(($(date +%s%N) / 1000000))

set +e
body="$(curl -sS --max-time "$TIMEOUT_SECS" -w '\n%{http_code}' "$API_URL/api/health" 2>/tmp/check-status-err)"
curl_exit=$?
set -e

end_ms=$(($(date +%s%N) / 1000000))
latency_ms=$((end_ms - start_ms))

if [ "$curl_exit" -ne 0 ]; then
    record=$(jq -nc \
        --arg checked_at "$checked_at" \
        --argjson latency_ms "$latency_ms" \
        --arg error "$(cat /tmp/check-status-err | tr -d '\n')" \
        '{checked_at: $checked_at, reachable: false, database: false, discord: false, latency_ms: $latency_ms, error: $error}')
else
    http_status="$(echo "$body" | tail -n1)"
    json_body="$(echo "$body" | sed '$d')"

    database="$(echo "$json_body" | jq -r '.database // false' 2>/dev/null || echo false)"
    discord="$(echo "$json_body" | jq -r '.discord // false' 2>/dev/null || echo false)"

    record=$(jq -nc \
        --arg checked_at "$checked_at" \
        --argjson latency_ms "$latency_ms" \
        --argjson http_status "$http_status" \
        --argjson database "$database" \
        --argjson discord "$discord" \
        '{checked_at: $checked_at, reachable: true, http_status: $http_status, database: $database, discord: $discord, latency_ms: $latency_ms}')
fi

echo "$record" >> "$HISTORY_FILE"

if [ "$(wc -l < "$HISTORY_FILE")" -gt "$MAX_LINES" ]; then
    tail -n "$MAX_LINES" "$HISTORY_FILE" > "$HISTORY_FILE.tmp"
    mv "$HISTORY_FILE.tmp" "$HISTORY_FILE"
fi

echo "Recorded: $record"
