

picks = {}
for o in obs:
    ob, net, sec, xs = convert(o)
    kind = None
    if sec == "reality" and net in ("tcp", "raw"):
        kind = "reality"
    elif net in ("xhttp", "splithttp"):
        kind = "xhttp"
    if kind and kind not in picks:
        picks[kind] = (ob, net, sec, (xs or {}).get("mode", "-"), o.get("tag", "-"))
for kind, (ob, net, sec, mode, tag) in picks.items():
    cfg = {"log": {"level": "warn"}, "outbounds": [ob]}
    cfg["inbounds"] = [{"type": "mixed", "listen": "127.0.0.1", "listen_port": 10808}]
    with open(f"e2e/{kind}.json", "w") as f:
        json.dump(cfg, f)
    print(f"{kind}: tag={tag} net={net} sec={sec} mode={mode} flow={ob.get('flow', '-')}")
for kind in ("reality", "xhttp"):
    if kind not in picks:
        print(f"{kind}: NOT FOUND in subscription")
