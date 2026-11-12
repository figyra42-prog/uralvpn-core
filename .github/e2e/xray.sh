#!/usr/bin/env bash
# UralVPN e2e control: official Xray-core with the same xhttp outbound.
set -u
python3 .github/e2e/xray_cfg.py
[ -f e2e/xray-xhttp.json ] || exit 0
mkdir -p e2e/xray
curl -sSL -o e2e/xray.zip https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip
unzip -q -o e2e/xray.zip -d e2e/xray
./e2e/xray/xray version | head -1
./e2e/xray/xray run -c e2e/xray-xhttp.json > e2e/xray.log 2>&1 &
pid=$!
sleep 3
ok=0
for url in https://www.google.com/generate_204 https://ifconfig.me/ip; do
  res=$(curl -s -o /dev/null -w '%{http_code} %{time_total}s' -m 20 -x socks5h://127.0.0.1:10809 "$url")
  echo "xray-xhttp ${url#https://}: $res"
  case "$res" in 200*|204*) ok=1;; esac
done
kill "$pid" 2>/dev/null; wait "$pid" 2>/dev/null
if [ "$ok" = 1 ]; then echo "xray-xhttp: OK"; else echo "xray-xhttp: FAIL, log tail:"; tail -n 15 e2e/xray.log; fi
