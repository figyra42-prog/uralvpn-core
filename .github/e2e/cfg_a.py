#!/usr/bin/env python3
# UralVPN e2e step 2: Xray vless outbounds -> sing-box configs (one reality, one xhttp).
import json


def mask(v):
    if v and isinstance(v, (str, int)) and len(str(v)) > 3:
        print(f"::add-mask::{v}", flush=True)


obs = json.load(open("e2e/outbounds.json"))


def server_of(o):
    s = o.get("settings", {})
    if s.get("vnext"):
        v = s["vnext"][0]
        u = v["users"][0]
        return v.get("address"), v.get("port", 443), u.get("id"), u.get("flow", "")
    return s.get("address"), s.get("port", 443), s.get("id"), s.get("flow", "")
