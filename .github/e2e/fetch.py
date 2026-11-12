#!/usr/bin/env python3
# UralVPN e2e step 1: fetch subscription, save vless links. Prints only counts and names.
import base64, os, urllib.parse as up, urllib.request as ur

sub = os.environ["SUB_URL"]
print(f"::add-mask::{sub}", flush=True)
best = []
for ua in ("v2rayNG/1.10.0", "Happ/3.5.0", "v2rayN/7.0"):
    req = ur.Request(sub, headers={"User-Agent": ua})
    with ur.urlopen(req, timeout=30) as r:
        raw = r.read().decode()
    try:
        dec = base64.b64decode(raw + "=" * (-len(raw) % 4)).decode()
        if "://" in dec:
            raw = dec
    except Exception:
        pass
    found = [l.strip() for l in raw.splitlines() if l.strip().startswith("vless://")]
    names = [up.unquote(l.split("#", 1)[1]) if "#" in l else "-" for l in found]
    print(f"UA {ua}: bytes={len(raw)} vless={len(found)} names={names[:40]}")
    if len(found) > len(best):
        best = found
os.makedirs("e2e", exist_ok=True)
with open("e2e/links.txt", "w") as f:
    f.write("\n".join(best))
print("saved vless links:", len(best))
