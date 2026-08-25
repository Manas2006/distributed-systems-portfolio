#!/usr/bin/env bash
set -euo pipefail

DEMO_DIR="$(mktemp -d /tmp/streamforge.XXXXXX)"
PIDS=()
cleanup() {
  for pid in "${PIDS[@]}"; do kill "$pid" 2>/dev/null || true; done
}
trap cleanup EXIT INT TERM

ffmpeg -hide_banner -loglevel error -y \
  -f lavfi -i testsrc2=size=1920x1080:rate=30 \
  -f lavfi -i sine=frequency=1000 -t 12 \
  -c:v libx264 -c:a aac -shortest "$DEMO_DIR/sample.mp4"

go run ./cmd/video-api -listen :6060 -data "$DEMO_DIR/data" >"$DEMO_DIR/api.log" 2>&1 &
PIDS+=("$!")
go run ./cmd/video-worker -id demo-worker -api http://127.0.0.1:6060 >"$DEMO_DIR/worker.log" 2>&1 &
PIDS+=("$!")
for _ in $(seq 1 60); do
  curl -fsS http://127.0.0.1:6060/healthz >/dev/null 2>&1 && break
  sleep 0.25
done

UPLOAD_ID="$(curl -fsS -X POST http://127.0.0.1:6060/v1/uploads | sed -n 's/.*"upload_id":"\([^"]*\)".*/\1/p')"
curl -fsS -X PATCH "http://127.0.0.1:6060/v1/uploads/$UPLOAD_ID" \
  -H 'Upload-Offset: 0' --data-binary "@$DEMO_DIR/sample.mp4"
curl -fsS -X POST "http://127.0.0.1:6060/v1/uploads/$UPLOAD_ID/complete"
for _ in $(seq 1 120); do
  status="$(curl -fsS "http://127.0.0.1:6060/v1/jobs/$UPLOAD_ID")"
  echo "$status"
  echo "$status" | grep -q '"state":"completed"' && exit 0
  echo "$status" | grep -q '"state":"dead"' && exit 1
  sleep 1
done
echo "video job did not complete within 120 seconds" >&2
exit 1

