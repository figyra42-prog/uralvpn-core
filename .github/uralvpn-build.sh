#!/usr/bin/env bash
# UralVPN: build sing-box core (linux-amd64 for tests, android-arm64 for the app)
set -euo pipefail
LDF="$(cat release/LDFLAGS) -s -w -buildid="
mkdir -p dist
for t in linux-amd64 android-arm64; do
  echo "=== build $t"
  GOOS="${t%-*}" GOARCH="${t#*-}" go build -trimpath -ldflags "$LDF" -tags "$TAGS" -o "dist/sing-box-$t" ./cmd/sing-box
done
ls -la dist
echo "=== version"
./dist/sing-box-linux-amd64 version
echo "=== xhttp sources"
grep -rlI --include=*.go -i xhttp . | head -5 || true
echo "=== key_share in options"
grep -rnI --include=*.go key_share option/ | head -5 || true
