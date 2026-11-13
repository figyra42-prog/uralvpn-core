package uralvpn

import (
	"encoding/json"
)

// WithTLSFragment turns on the ClientHello fragmentation (Xray's finalMask fragment "tlshello",
// 10-20 bytes, no delay) on the direct Reality servers. The panel sets it on its direct Xray hosts
// to get past providers that cut the TLS handshake; the panel's sing-box profile has no such
// setting, so the servers work in the Xray apps but not here. Servers that already carry their own
// tls_fragment, the CDN (XHTTP) servers and the UDP ones (Hysteria2) are left alone.
func WithTLSFragment(config string) (string, error) {
	var parsed map[string]any
	if err := decode(config, &parsed); err != nil {
		return "", err
	}
	for _, item := range list(parsed["outbounds"]) {
		outbound := object(item)
		if outbound == nil || text(outbound["type"]) != "vless" {
			continue
		}
		reality := object(object(outbound["tls"])["reality"])
		if reality == nil {
			continue
		}
		if enabled, ok := reality["enabled"].(bool); ok && !enabled {
			continue
		}
		if transport := object(outbound["transport"]); transport != nil {
			if kind := text(transport["type"]); kind == "xhttp" || kind == "splithttp" {
				continue
			}
		}
		if _, set := outbound["tls_fragment"]; set {
			continue
		}
		outbound["tls_fragment"] = map[string]any{"enabled": true, "size": "10-20", "sleep": "0"}
		// fragmentation does not work together with TCP Fast Open
		outbound["tcp_fast_open"] = false
	}
	content, err := json.Marshal(parsed)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
