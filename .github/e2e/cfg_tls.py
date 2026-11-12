

def tls_part(ob, ss, sec, addr):
    if sec not in ("tls", "reality"):
        return
    key = "realitySettings" if sec == "reality" else "tlsSettings"
    ts = ss.get(key, {})
    sni = ts.get("serverName") or addr
    mask(sni)
    fp = ts.get("fingerprint") or "chrome"
    t = {"enabled": True, "server_name": sni}
    t["utls"] = {"enabled": True, "fingerprint": fp}
    if ts.get("alpn"):
        t["alpn"] = ts["alpn"]
    if sec == "reality":
        r = {"enabled": True}
        r["public_key"] = ts.get("publicKey", "")
        r["short_id"] = ts.get("shortId", "")
        t["reality"] = r
    ob["tls"] = t
