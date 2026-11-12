#!/usr/bin/env bash
set -euo pipefail
export CGO_ENABLED=0 GOTOOLCHAIN=local
TAGS=with_gvisor,with_quic,with_utls,with_grpc,badlinkname,tfogo_checklinkname0
mkdir -p dist
go build -trimpath -ldflags "$(cat release/LDFLAGS) -s -w -buildid=" -tags "$TAGS" -o dist/sing-box-linux-amd64 ./cmd/sing-box
ls -la dist
