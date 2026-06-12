#!/bin/bash
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

echo "== pinned Go toolchain =="
./tools/check-go-version.sh

echo "== gofmt check =="
TMP=$(mktemp)
gofmt -l server client cmd > "$TMP"
if [ -s "$TMP" ]; then cat "$TMP"; echo "gofmt differences found" >&2; exit 1; fi
rm -f "$TMP"

echo "== go test =="
go test ./...

echo "== go vet =="
go vet ./...

echo "== static security checks =="
python3 tests/security/check_static_invariants.py

echo "== govulncheck =="
if ! command -v govulncheck >/dev/null 2>&1; then
  echo "govulncheck is required for production release checks" >&2
  echo "Install with: go install golang.org/x/vuln/cmd/govulncheck@latest" >&2
  exit 1
fi
govulncheck ./...

echo "== build binaries =="
./tools/build-binaries.sh

echo "== backend smoke test =="
DB=$(mktemp /tmp/deaddrop-smoke.XXXXXX.db)
LOG=$(mktemp /tmp/deaddrop-smoke.XXXXXX.log)
./artifacts/deaddrop-server -db "$DB" > "$LOG" 2>&1 &
PID=$!
cleanup(){ kill "$PID" >/dev/null 2>&1 || true; rm -f "$DB" "$DB"-* "$LOG"; }
trap cleanup EXIT
for _ in $(seq 1 50); do curl -fsS http://127.0.0.1:8080/health >/dev/null && break || sleep 0.1; done
PAYLOAD=$(python3 - <<'PY'
import base64
print(base64.urlsafe_b64encode(b'{"version":"1"}').decode().rstrip('='))
PY
)
RESP=$(curl -fsS -X POST http://127.0.0.1:8080/drop -H 'Content-Type: application/json' -d "{\"hint\":\"release_smoke\",\"payload\":\"$PAYLOAD\"}")
ID=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])' <<<"$RESP")
curl -fsS 'http://127.0.0.1:8080/drops?hint=release_smoke' | grep -q "$ID"
[ "$(curl -sS -o /dev/null -w '%{http_code}' -X DELETE "http://127.0.0.1:8080/drop/$ID")" = "204" ]
if grep -q 'release_smoke\|eyJ2ZXJzaW9u\|User-Agent\|RemoteAddr' "$LOG"; then
  echo "prohibited metadata found in logs" >&2
  cat "$LOG" >&2
  exit 1
fi

echo "release checks passed"
