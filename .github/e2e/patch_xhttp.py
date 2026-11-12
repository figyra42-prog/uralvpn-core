#!/usr/bin/env python3
# UralVPN: patch the hiddify-based core so that it accepts and measures like Xray-core.
# 1) transport/v2rayxhttp/client.go: scMaxEachPostBytes only has to be > 0 and the upload pipe
#    limit is max(0, size - buf.Size), as in Xray-core splithttp/dialer.go. Panel hosts use
#    1536-6144 (smaller than one 8 KiB buffer); the older check panicked and took the core down.
# 2) option/v2ray_transport.go: allow seq in cookie/header/query while the session id is in the path.
#    Xray-core has no such rule and ApplyMetaToRequest places both independently; 8 of 11 panel
#    CDN hosts use this combination and were rejected as invalid.
# 3) common/urltest/urltest.go: always measure like Xray's MeasureDelay: two requests over one
#    connection, the faster one counts (the second skips the TCP/TLS handshake). The core already
#    has this "unified delay" mode, but only when the context asks for it; nothing did.
# Usage (from the core root): patch_xhttp.py transport/v2rayxhttp/client.go
import os
import sys


def patch(path, pairs, name):
    s = open(path).read()
    if all(new in s for _, new in pairs):
        print(name + ": already patched")
        return
    for old, new in pairs:
        assert s.count(old) == 1, name + ": not found: " + old.splitlines()[0]
        s = s.replace(old, new)
    open(path, "w").write(s)
    print(name + ": patched")


client = sys.argv[1]
patch(client, [
    ("""\tif scMaxEachPostBytes.From <= buf.Size {
\t\tpanic("`scMaxEachPostBytes` should be bigger than " + strconv.Itoa(buf.Size))
\t}""", """\tif scMaxEachPostBytes.From <= 0 { //UralVPN: as Xray-core, small posts are allowed
\t\treturn fail(E.New("xhttp: sc_max_each_post_bytes should be bigger than 0"))
\t}"""),
    ("pipe.New(pipe.WithSizeLimit(maxUploadSize - buf.Size))",
     "pipe.New(pipe.WithSizeLimit(max(0, maxUploadSize-buf.Size))) //UralVPN: as Xray-core"),
], "xhttp client")

root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(client))))
patch(os.path.join(root, "option", "v2ray_transport.go"), [
    ("""\tcase "cookie", "header", "query":
\t\tif options.SessionPlacement == "path" {
\t\t\treturn E.New("SeqPlacement must be path when SessionPlacement is path")
\t\t}""", """\tcase "cookie", "header", "query": //UralVPN: as Xray-core, seq may leave the path alone"""),
], "xhttp options")

patch(os.path.join(root, "common", "urltest", "urltest.go"), [
    ("\tif IsUnifiedDelayFromContext(ctx) {\n",
     "\tif true { //UralVPN: always the unified delay (two requests, the faster counts), like Xray's MeasureDelay\n"),
    ("""\t\tresp, err = client.Do(req.WithContext(ctx)) //H: a cancel did not stop the second request
\t\tif err != nil {
\t\t\treturn
\t\t}
\t\tresp.Body.Close()
\t\tt = uint16(time.Since(second) / time.Millisecond) //to avid timeout in the second call""",
     """\t\tresp, err = client.Do(req.WithContext(ctx)) //H: a cancel did not stop the second request
\t\tif err != nil {
\t\t\terr = nil //UralVPN: the first request worked, keep its delay
\t\t\treturn
\t\t}
\t\tresp.Body.Close()
\t\tif delay := uint16(time.Since(second) / time.Millisecond); delay < t {
\t\t\tt = delay
\t\t}"""),
], "urltest")
