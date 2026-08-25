#!/usr/bin/env bash
set -euo pipefail

DEMO_DIR="$(mktemp -d /tmp/atlas-search.XXXXXX)"
PIDS=()
cleanup() {
  for pid in "${PIDS[@]}"; do kill "$pid" 2>/dev/null || true; done
}
trap cleanup EXIT INT TERM

for port in 8081 8082 8083; do
  go run ./cmd/search-node -listen ":$port" -data "$DEMO_DIR/$port" >"$DEMO_DIR/$port.log" 2>&1 &
  PIDS+=("$!")
done
go run ./cmd/search-coordinator -listen :8080 \
  -shards 'a=http://127.0.0.1:8081|http://127.0.0.1:8082|http://127.0.0.1:8083' \
  >"$DEMO_DIR/coordinator.log" 2>&1 &
PIDS+=("$!")

for _ in $(seq 1 60); do
  curl -fsS http://127.0.0.1:8080/healthz >/dev/null 2>&1 && break
  sleep 0.25
done
curl -fsS -X POST http://127.0.0.1:8080/v1/documents \
  -H 'content-type: application/json' \
  -d '{"id":"paper-1","title":"Raft consensus","body":"replicated logs and leader election"}'
curl -fsS -X POST http://127.0.0.1:8080/v1/documents \
  -H 'content-type: application/json' \
  -d '{"id":"paper-2","title":"Search indexing","body":"compressed postings and BM25 ranking"}'
curl -fsS 'http://127.0.0.1:8080/v1/search?q=replicated+leader&limit=5'

