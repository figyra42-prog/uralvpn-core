#!/usr/bin/env python3
# UralVPN e2e step 2: vless links -> sing-box configs (one reality, one xhttp).
import json, os, urllib.parse as up


def mask(v):
    if v:
        print(f"::add-mask::{v}", flush=True)


links = [l for l in open("e2e/links.txt").read().splitlines() if l]


def parse(l):
    u = up.urlsplit(l)
    q = {k: v[0] for k, v in up.parse_qs(u.query).items()}
    for v in (u.username, u.hostname, q.get("pbk"), q.get("sid"), q.get("sni"), q.get("host"), q.get("path")):
        mask(v)
    return u, q
