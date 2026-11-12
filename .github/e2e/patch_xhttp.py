#!/usr/bin/env python3
# UralVPN: make the xhttp transport accept what current Xray-core accepts (the panel's CDN hosts).
# 1) transport/v2rayxhttp/client.go: scMaxEachPostBytes only has to be > 0 and the upload pipe
#    limit is max(0, size - buf.Size), as in Xray-core splithttp/dialer.go. Panel hosts use
#    1536-6144 (smaller than one 8 KiB buffer); the older check panicked and took the core down.
# 2) option/v2ray_transport.go: allow seq in cookie/header/query while the session id is in the path.
#    Xray-core has no such rule and ApplyMetaToRequest places both independently; 8 of 11 panel
#    CDN hosts use this combination and were rejected as invalid.
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
