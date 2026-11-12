package libbox

import "github.com/sagernet/sing-box/common/uralvpn"

// UralvpnMergeSubscription returns the UralVPN config: the panel's sing-box profile plus the CDN
// servers from its Xray JSON subscription, with the direct -> CDN autopilot. //UralVPN
func UralvpnMergeSubscription(singboxConfig string, xrayConfig string) (string, error) {
	return uralvpn.Merge(singboxConfig, xrayConfig)
}
