package uralvpn

import (
	"encoding/json"
	"regexp"
	"strings"
)

var suffixRegexp = regexp.MustCompile(`^\\\.([A-Za-z0-9.\-]+|[^\x00-\x7f]+)\$$`)

// directRules turns the panel's Xray routing ("direct" for Russian addresses and domains) into
// sing-box route rules, so that Russian sites do not go through the VPN, like in the Xray apps.
// It returns nil when the Xray config has no such rules.
func directRules(xrayConfigs []map[string]any, directTag string) []any {
	var cidrs, suffixes, regexes []string
	seenCIDR, seenSuffix, seenRegex := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, xray := range xrayConfigs {
		for _, item := range list(object(xray["routing"])["rules"]) {
			rule := object(item)
			if rule == nil || text(rule["outboundTag"]) != "direct" {
				continue
			}
			for _, entry := range list(rule["ip"]) {
				value := strings.TrimSpace(text(entry))
				if value == "" || strings.Contains(value, ":") && !strings.Contains(value, "/") || strings.HasPrefix(value, "geoip:") {
					continue
				}
				if !seenCIDR[value] {
					seenCIDR[value] = true
					cidrs = append(cidrs, value)
				}
			}
			for _, entry := range list(rule["domain"]) {
				value := strings.TrimSpace(text(entry))
				switch {
				case value == "" || strings.HasPrefix(value, "geosite:") || strings.HasPrefix(value, "ext:"):
				case strings.HasPrefix(value, "regexp:"):
					pattern := strings.TrimPrefix(value, "regexp:")
					if match := suffixRegexp.FindStringSubmatch(pattern); match != nil {
						if suffix := strings.ToLower(match[1]); !seenSuffix[suffix] {
							seenSuffix[suffix] = true
							suffixes = append(suffixes, suffix)
						}
					} else if !seenRegex[pattern] {
						seenRegex[pattern] = true
						regexes = append(regexes, pattern)
					}
				default:
					value = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(value, "domain:"), "full:"), "keyword:")
					if suffix := strings.ToLower(value); suffix != "" && !seenSuffix[suffix] {
						seenSuffix[suffix] = true
						suffixes = append(suffixes, suffix)
					}
				}
			}
		}
		if len(cidrs) > 0 || len(suffixes) > 0 {
			break // every config of the subscription carries the same routing
		}
	}
	var rules []any
	if len(suffixes) > 0 {
		rules = append(rules, map[string]any{"action": "route", "outbound": directTag, "domain_suffix": suffixes})
	}
	if len(regexes) > 0 {
		rules = append(rules, map[string]any{"action": "route", "outbound": directTag, "domain_regex": regexes})
	}
	if len(cidrs) > 0 {
		rules = append(rules, map[string]any{"action": "route", "outbound": directTag, "ip_cidr": cidrs})
	}
	return rules
}

// insertDirectRules puts the rules after the leading action rules (sniff, hijack-dns, ...) of the
// route, so they are checked before the final selector but after DNS handling.
func insertDirectRules(route map[string]any, rules []any) {
	if len(rules) == 0 {
		return
	}
	existing := list(route["rules"])
	index := 0
	for index < len(existing) {
		rule := object(existing[index])
		action := text(rule["action"])
		if rule == nil || action == "" || action == "route" || rule["outbound"] != nil {
			break
		}
		index++
	}
	merged := make([]any, 0, len(existing)+len(rules))
	merged = append(merged, existing[:index]...)
	merged = append(merged, rules...)
	merged = append(merged, existing[index:]...)
	route["rules"] = merged
}

// WithDirectRules adds the Russian "direct" rules of the Xray subscription to the merged config.
func WithDirectRules(merged string, xrayConfig string) (string, error) {
	var config map[string]any
	if err := decode(merged, &config); err != nil {
		return "", err
	}
	var xrayConfigs []map[string]any
	if err := decode(xrayConfig, &xrayConfigs); err != nil {
		var single map[string]any
		if err2 := decode(xrayConfig, &single); err2 != nil {
			return "", err
		}
		xrayConfigs = []map[string]any{single}
	}
	directTag := "direct"
	for _, item := range list(config["outbounds"]) {
		if outbound := object(item); outbound != nil && text(outbound["type"]) == "direct" && text(outbound["tag"]) != "" {
			directTag = text(outbound["tag"])
			break
		}
	}
	rules := directRules(xrayConfigs, directTag)
	if len(rules) == 0 {
		return merged, nil
	}
	route := object(config["route"])
	if route == nil {
		route = map[string]any{}
		config["route"] = route
	}
	insertDirectRules(route, rules)
	content, err := json.Marshal(config)
	return string(content), err
}
