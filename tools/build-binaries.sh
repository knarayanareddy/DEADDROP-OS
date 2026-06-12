#!/bin/bash
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
OUT=${OUT:-$ROOT/artifacts}
mkdir -p "$OUT"
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$OUT/deaddrop-server" ./cmd/deaddrop-server
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$OUT/deaddrop-client" ./cmd/deaddrop-client
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$OUT/deaddrop-panic" ./cmd/deaddrop-panic
(cd "$OUT" && sha256sum deaddrop-* > SHA256SUMS)
echo "Binaries written to $OUT"
echo "ISO assembly uses Debian live-build; see iso/deaddropos.conf and iso/live-config."
