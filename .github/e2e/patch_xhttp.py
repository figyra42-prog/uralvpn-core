#!/usr/bin/env python3
# UralVPN: patch transport/v2rayxhttp/client.go like current Xray-core (splithttp/dialer.go):
# scMaxEachPostBytes only has to be > 0, the upload pipe limit is max(0, size - buf.Size).
# Panel CDN hosts use scMaxEachPostBytes 1536-6144 (smaller than one 8 KiB buffer), which the
# older check rejected with a panic that took the whole core down.
import sys

path = sys.argv[1]
s = open(path).read()
old_check = """\tif scMaxEachPostBytes.From <= buf.Size {
\t\tpanic("`scMaxEachPostBytes` should be bigger than " + strconv.Itoa(buf.Size))
\t}"""
new_check = """\tif scMaxEachPostBytes.From <= 0 { //UralVPN: as Xray-core, small posts are allowed
\t\treturn fail(E.New("xhttp: sc_max_each_post_bytes should be bigger than 0"))
\t}"""
old_pipe = "pipe.New(pipe.WithSizeLimit(maxUploadSize - buf.Size))"
new_pipe = "pipe.New(pipe.WithSizeLimit(max(0, maxUploadSize-buf.Size))) //UralVPN: as Xray-core"
if new_check in s and new_pipe in s:
    print("xhttp client: already patched")
    sys.exit(0)
assert s.count(old_check) == 1, "check not found"
assert s.count(old_pipe) == 1, "pipe not found"
s = s.replace(old_check, new_check).replace(old_pipe, new_pipe)
open(path, "w").write(s)
print("xhttp client: patched")
