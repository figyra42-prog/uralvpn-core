

def xhttp_part(ob, xs):
    mask(xs.get("host"))
    mask(xs.get("path"))
    ex = dict(xs.get("extra") or {})
    for k, v in xs.items():
        if k != "extra":
            ex[k] = v
    print("xhttp keys:", sorted(xs.keys()), flush=True)
    print("extra keys:", sorted((xs.get("extra") or {}).keys()), flush=True)
    print("xmux keys:", sorted((ex.get("xmux") or {}).keys()), flush=True)
    tr = {"type": "xhttp"}
    tr["mode"] = xs.get("mode") or "auto"
    tr["path"] = xs.get("path") or "/"
    if xs.get("host"):
        tr["host"] = xs["host"]
    for k, sk in XMAP.items():
        if k in ex:
            tr[sk] = rng(ex[k])
    hd = ex.get("headers") or {}
    hd = {k: v for k, v in hd.items() if k.lower() != "host"}
    if hd:
        tr["headers"] = hd
    mux = {}
    for k, v in (ex.get("xmux") or {}).items():
        if k in MUXMAP:
            mux[MUXMAP[k]] = rng(v)
    if mux:
        tr["xmux"] = mux
    ob["transport"] = tr
