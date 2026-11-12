

picks = {}
for l in links:
    u, q = parse(l)
    t = q.get("type", "tcp")
    kind = "reality" if (q.get("security") == "reality" and t == "tcp") else ("xhttp" if t == "xhttp" else None)
    if kind and kind not in picks:
        name = up.unquote(l.split("#", 1)[1]) if "#" in l else "-"
        picks[kind] = (u, q, name)
for kind, (u, q, name) in picks.items():
    cfg = {"log": {"level": "warn"},
           "inbounds": [{"type": "mixed", "listen": "127.0.0.1", "listen_port": 10808}],
           "outbounds": [outbound(u, q)]}
    with open(f"e2e/{kind}.json", "w") as f:
        json.dump(cfg, f)
    print(f"{kind}: [{name}] type={q.get('type', 'tcp')} security={q.get('security')} mode={q.get('mode', '-')} flow={q.get('flow', '-')} fp={q.get('fp', '-')}")
for kind in ("reality", "xhttp"):
    if kind not in picks:
        print(f"{kind}: NOT FOUND in subscription")
