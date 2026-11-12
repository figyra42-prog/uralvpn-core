

def xhttp_part(ob, xs):
    mask(xs.get("host"))
    mask(xs.get("path"))
    ex = dict(xs.get("extra") or {})
    for k, v in xs.items():
        if k != "extra":
            ex[k] = v
    tr = {"type": "xhttp"}
    tr["mode"] = xs.get("mode") or "auto"
    tr["path"] = xs.get("path") or "/"
    if xs.get("host"):
        tr["host"] = xs["host"]
    for k, sk in XMAP.items():
        if k in ex:
            tr[sk] = rng(ex[k])
    for k, sk in INTS.items():
        if k in ex:
            tr[sk] = to_int(ex[k])
    hd = ex.get("headers") or {}
    hd = {k: v for k, v in hd.items() if k.lower() != "host"}
    if hd:
        tr["headers"] = hd
    mux = {}
    for k, v in (ex.get("xmux") or {}).items():
        if k == "hKeepAlivePeriod":
            mux[MUXMAP[k]] = to_int(v)
        elif k in MUXMAP:
            mux[MUXMAP[k]] = rng(v)
    if mux:
        tr["xmux"] = mux
    print("sing-box xhttp keys:", sorted(tr.keys()), flush=True)
    ob["transport"] = tr
