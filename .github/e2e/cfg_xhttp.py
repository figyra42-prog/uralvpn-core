

def xhttp_part(ob, xs):
    mask(xs.get("host"))
    mask(xs.get("path"))
    tr = {"type": "xhttp"}
    tr["mode"] = xs.get("mode") or "auto"
    tr["path"] = xs.get("path") or "/"
    if xs.get("host"):
        tr["host"] = xs["host"]
    ob["transport"] = tr
