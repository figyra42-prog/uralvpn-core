#!/usr/bin/env python3
# UralVPN e2e control: same first xhttp outbound, run through official Xray-core.
import json

obs = json.load(open("e2e/outbounds.json"))
pick = None
for o in obs:
    net = o.get("streamSettings", {}).get("network", "tcp")
    if net in ("xhttp", "splithttp"):
        pick = dict(o)
        break
if pick is None:
    print("xray-xhttp: NOT FOUND")
    raise SystemExit(0)
print("xray control outbound tag:", pick.get("tag"))
pick["tag"] = "proxy"
ss = pick.get("streamSettings", {})
so = ss.get("sockopt") or {}
so.pop("dialerProxy", None)
if so:
    ss["sockopt"] = so
else:
    ss.pop("sockopt", None)
cfg = {"log": {"loglevel": "warning"}, "outbounds": [pick]}
cfg["inbounds"] = [{"protocol": "socks", "listen": "127.0.0.1", "port": 10809, "settings": {"udp": False}}]
json.dump(cfg, open("e2e/xray-xhttp.json", "w"))
