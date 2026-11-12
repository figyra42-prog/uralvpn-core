

XMAP = {}
XMAP["xPaddingBytes"] = "x_padding_bytes"
XMAP["noGRPCHeader"] = "no_grpc_header"
XMAP["scMaxEachPostBytes"] = "sc_max_each_post_bytes"
XMAP["scMinPostsIntervalMs"] = "sc_min_posts_interval_ms"
MUXMAP = {}
MUXMAP["maxConcurrency"] = "max_concurrency"
MUXMAP["maxConnections"] = "max_connections"
MUXMAP["cMaxReuseTimes"] = "c_max_reuse_times"
MUXMAP["hMaxRequestTimes"] = "h_max_request_times"
MUXMAP["hMaxReusableSecs"] = "h_max_reusable_secs"
MUXMAP["hKeepAlivePeriod"] = "h_keep_alive_period"


def rng(v):
    if isinstance(v, dict) and "from" in v:
        return {"from": v["from"], "to": v.get("to", v["from"])}
    return v
