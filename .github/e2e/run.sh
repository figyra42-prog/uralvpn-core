#!/usr/bin/env bash
# UralVPN e2e: run sing-box with each config and fetch test URLs through it.
set -u
SB=./dist/sing-box-linux-amd64
fail=0
for kind in reality xhttp; do
  cfg="e2e/$kind.json"
  if [ ! -f "$cfg" ]; then echo "$kind: NO CONFIG"; fail=1; continue; fi
  "$SB" run -c "$cfg" > "e2e/$kind.log" 2>&1 &
  pid=$!
  sleep 3
  ok=0
  for url in https://www.google.com/generate_204 https://ifconfig.me/ip https://www.cloudflare.com/cdn-cgi/trace; do
    res=$(curl -s -o /dev/null -w '%{http_code} %{time_total}s' -m 20 -x socks5h://127.0.0.1:10808 "$url")
    echo "$kind ${url#https://}: $res"
    case "$res" in 200*|204*) ok=1;; esac
  done
  kill "$pid" 2>/dev/null; wait "$pid" 2>/dev/null
  if [ "$ok" = 1 ]; then
    echo "$kind: OK"
  else
    echo "$kind: FAIL, core log tail:"; tail -n 20 "e2e/$kind.log"; fail=1
  fi
done
exit $fail
