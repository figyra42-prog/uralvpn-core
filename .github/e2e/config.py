#!/usr/bin/env python3
# UralVPN e2e: subscription -> sing-box configs (one reality, one xhttp). No secrets printed.
import base64, json, os, urllib.parse as up, urllib.request as ur


def mask(v):
    if v:
        print(f"::add-mask::{v}", flush=True)


sub = os.environ["SUB_URL"]
mask(sub)
req = ur.Request(sub, headers={"User-Agent": "v2rayNG/1.10.0"})
raw = ur.urlopen(req, timeout=30).read().decode()
try:
    raw = base64.b64decode(raw + "=" * (-len(raw) % 4)).decode()
except Exception:
    pass
links = [l.strip() for l in raw.splitlines() if l.startswith("vless://")]
print("vless links:", len(links))


def parse(l):
    u = up.urlsplit(l)
    q = {k: v[0] for k, v in up.parse_qs(u.query).items()}
    for v in (u.username, u.hostname, q.get("pbk"), q.get("sid"), q.get("sni"), q.get("host"), q.get("path")):
        mask(v)
    return u, q


def outbound(u, q):
    o = {"type": "vless", "tag": "p", "server": u.hostname,
         "server_port": u.port or 443, "uuid": u.username}
    if q.get("flow"):
        o["flow"] = q["flow"]
    sec = q.get("security", "none")
    if sec in ("tls", "reality"):
        t = {"enabled": True, "server_name": q.get("sni") or u.hostname,
             "utls": {"enabled": True, "fingerprint": q.get("fp") or "chrome"}}
        if q.get("alpn"):
            t["alpn"] = q["alpn"].split(",")
        if sec == "reality":
            t["reality"] = {"enabled": True, "public_key": q.get("pbk", ""),
                            "short_id": q.get("sid", "")}
        o["tls"] = t
    if q.get("type") == "xhttp":
        tr = {"type": "xhttp", "mode": q.get("mode") or "auto", "path": q.get("path") or "/"}
        if q.get("host"):
            tr["host"] = q["host"]
        o["transport"] = tr
    return o


picks = {}
for l in links:
    u, q = parse(l)
    t = q.get("type", "tcp")
    kind = "reality" if (q.get("security") == "reality" and t == "tcp") else ("xhttp" if t == "xhttp" else None)
    if kind and kind not in picks:
        picks[kind] = (u, q)
os.makedirs("e2e", exist_ok=True)
for kind, (u, q) in picks.items():
    cfg = {"log": {"level": "warn"},
           "inbounds": [{"type": "mixed", "listen": "127.0.0.1", "listen_port": 10808}],
           "outbounds": [outbound(u, q)]}
    with open(f"e2e/{kind}.json", "w") as f:
        json.dump(cfg, f)
    print(f"{kind}: type={q.get('type', 'tcp')} security={q.get('security')} mode={q.get('mode', '-')} flow={q.get('flow', '-')} fp={q.get('fp', '-')}")
for kind in ("reality", "xhttp"):
    if kind not in picks:
        print(f"{kind}: NOT FOUND in subscription")
