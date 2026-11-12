package uralvpn

import (
	"encoding/json"
)

// WithRealityKeyShare sets key_share "auto" on every Reality outbound: the TLS hello with the
// post-quantum key share (hybrid, the core's default) is tried first, and the classical X25519
// one is used when hybrid gets no answer. The larger hello is a suspect for servers that answer
// the old Xray client but not this core; "auto" keeps the hybrid for the servers where it works.
func WithRealityKeyShare(config string) (string, error) {
	var parsed map[string]any
	if err := decode(config, &parsed); err != nil {
		return "", err
	}
	for _, item := range list(parsed["outbounds"]) {
		tls := object(object(item)["tls"])
		reality := object(tls["reality"])
		if reality == nil {
			continue
		}
		if _, set := reality["key_share"]; !set {
			reality["key_share"] = "auto"
		}
	}
	content, err := json.Marshal(parsed)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
