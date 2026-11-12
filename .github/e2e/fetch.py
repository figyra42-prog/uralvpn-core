#!/usr/bin/env python3
# UralVPN e2e step 1: fetch subscription (Xray JSON) as a real client, save vless outbounds.
import json, os, urllib.request as ur

sub = os.environ["SUB_URL"]
print(f"::add-mask::{sub}", flush=True)
HDR = {"User-Agent": "Happ/3.5.0", "x-hwid": "uralvpn-e2e-ci", "x-device-os": "Android", "x-ver-os": "14", "x-device-model": "uralvpn-e2e"}
with ur.urlopen(ur.Request(sub, headers=HDR), timeout=30) as r:
    data = json.loads(r.read().decode())
configs = data if isinstance(data, list) else [data]
seen, obs = set(), []
for c in configs:
    for o in c.get("outbounds", []):
        if o.get("protocol") != "vless":
            continue
        key = json.dumps(o.get("settings"), sort_keys=True) + json.dumps(o.get("streamSettings"), sort_keys=True)
        if key in seen:
            continue
        seen.add(key)
        obs.append(o)
print("configs:", len(configs), "unique vless outbounds:", len(obs))
os.makedirs("e2e", exist_ok=True)
json.dump(obs, open("e2e/outbounds.json", "w"))
