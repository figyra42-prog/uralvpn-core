

def convert(o):
    addr, port, uid, flow = server_of(o)
    ss = o.get("streamSettings", {})
    net, sec = ss.get("network", "tcp"), ss.get("security", "none")
    for v in (addr, uid):
        mask(v)
    ob = {"type": "vless", "tag": "p", "server": addr, "server_port": int(port), "uuid": uid}
    if flow:
        ob["flow"] = flow
    tls_part(ob, ss, sec, addr)
    xs = ss.get("xhttpSettings") or ss.get("splithttpSettings")
    if net in ("xhttp", "splithttp") and xs is not None:
        xhttp_part(ob, xs)
    return ob, net, sec, xs
