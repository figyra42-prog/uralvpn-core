package libbox

import "github.com/sagernet/sing-box/common/uralvpn"

// UralvpnMergeSubscription returns the UralVPN config: the panel's sing-box profile plus the CDN
// servers from its Xray JSON subscription, with the direct -> CDN autopilot. //UralVPN
func UralvpnMergeSubscription(singboxConfig string, xrayConfig string) (string, error) {
	merged, err := uralvpn.Merge(singboxConfig, xrayConfig)
	if err != nil {
		return "", err
	}
	// Russian sites go directly, as in the Xray apps (rules come from the Xray subscription)
	if withDirect, err := uralvpn.WithDirectRules(merged, xrayConfig); err == nil {
		merged = withDirect
	}
	// The direct servers get the same ClientHello fragmentation as in the panel's Xray hosts
	if withFragment, err := uralvpn.WithTLSFragment(merged); err == nil {
		merged = withFragment
	}
	if withKeyShare, err := uralvpn.WithRealityKeyShare(merged); err == nil {
		return withKeyShare, nil
	}
	return merged, nil
}

// UralvpnDescriptions returns JSON {"server": "description"} from the Xray subscription. //UralVPN
func UralvpnDescriptions(xrayConfig string) (string, error) {
	return uralvpn.Descriptions(xrayConfig)
}
