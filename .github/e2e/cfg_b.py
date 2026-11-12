

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
