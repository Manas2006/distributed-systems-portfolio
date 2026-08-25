#!/usr/bin/env bash
set -euo pipefail

DEMO_DIR="$(mktemp -d /tmp/chronos-tsdb.XXXXXX)"
go run ./cmd/tsdb -listen :5050 -data "$DEMO_DIR" -block-size 3 >"$DEMO_DIR/server.log" 2>&1 &
SERVER_PID="$!"
trap 'kill "$SERVER_PID" 2>/dev/null || true' EXIT INT TERM
for _ in $(seq 1 60); do
  curl -fsS http://127.0.0.1:5050/healthz >/dev/null 2>&1 && break
  sleep 0.25
done

curl -fsS -X POST http://127.0.0.1:5050/v1/write -H 'content-type: application/json' -d '[
  {"series":{"name":"request_latency_ms","labels":{"service":"search","region":"us-east"}},
   "samples":[{"timestamp":1000,"value":8.1},{"timestamp":2000,"value":7.8},{"timestamp":3000,"value":9.2}]}
]'
curl -fsS 'http://127.0.0.1:5050/v1/query?name=request_latency_ms&label.service=search&label.region=us-east&start=0&end=4000'

