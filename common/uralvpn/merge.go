// Package uralvpn builds the UralVPN client config: the panel's sing-box profile (DNS, route,
// TUN, direct servers) plus the CDN (XHTTP) servers that the panel only puts in its Xray JSON
// subscription, grouped like the panel's Xray "Two Tier" template: direct servers first, the CDN
// ones when no direct server connects, and back to direct ones once they work again.
package uralvpn

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// decode keeps numbers as written (json.Number): float64 would turn 1000000 into 1e+06,
// which sing-box range options do not read.
func decode(content string, value any) error {
	decoder := json.NewDecoder(bytes.NewReader([]byte(content)))
	decoder.UseNumber()
	return decoder.Decode(value)
}

const (
	TagSelector = "🌐 Локации"
	TagAuto     = "⚡ Автопилот"
	TagDirect   = "🚀 Прямые"
	TagCDN      = "☁️ CDN"

	testURL          = "https://www.gstatic.com/generate_204"
	testInterval     = "1m"
	blacklistTimeout = "30s"
	// The panel's profile leaves the monitor's own limit at the 5 s default, which is too tight on a
	// busy mobile network: a working server then shows as "no data". The group tests use their own
	// limit (see the app's urltest patch), this one covers the monitor.
	pingTestTimeout = "10s"
)

// Merge returns the sing-box config made from the panel's sing-box profile and Xray subscription.
func Merge(singboxConfig string, xrayConfig string) (string, error) {
	var config map[string]any
	if err := decode(singboxConfig, &config); err != nil {
		return "", fmt.Errorf("decode sing-box config: %w", err)
	}
	var xrayConfigs []map[string]any
	if err := decode(xrayConfig, &xrayConfigs); err != nil {
		var single map[string]any
		if err2 := decode(xrayConfig, &single); err2 != nil {
			return "", fmt.Errorf("decode xray config: %w", err)
		}
		xrayConfigs = []map[string]any{single}
	}

	// UralVPN: the panel lists its autopilot members itself (the Xray config with balancers):
	// the direct servers under "auto*" and the CDN servers under "cdn*"/"fb". Follow those lists
	// instead of taking every server of the profile: a location without the DIRECT tag stays a
	// manual one, and a CDN host without the CDN tag stays out of the CDN group.
	autoServers, cdnServers := panelAutopilotMembers(xrayConfigs)

	used := map[string]bool{}
	var directTags, manualTags []string
	var proxies, others []any
	for _, item := range list(config["outbounds"]) {
		outbound := object(item)
		if outbound == nil {
			continue
		}
		outboundType := text(outbound["type"])
		tag := text(outbound["tag"])
		switch outboundType {
		case "selector", "urltest", "fallback", "balancer":
			continue
		case "direct", "block", "dns":
			others = append(others, outbound)
			used[tag] = true
			continue
		}
		if isAutopilot(tag) {
			// the panel's "autopilot" entry is one of the direct servers under another name
			continue
		}
		tag = unique(used, tag)
		outbound["tag"] = tag
		proxies = append(proxies, outbound)
		if len(autoServers) > 0 && !autoServers[text(outbound["server"])] {
			manualTags = append(manualTags, tag)
			continue
		}
		directTags = append(directTags, tag)
	}
	if len(directTags) == 0 {
		return "", errors.New("no servers in sing-box config")
	}

	var cdnTags []string
	for _, xray := range xrayConfigs {
		if len(list(object(xray["routing"])["balancers"])) > 0 {
			// the Xray autopilot config: its servers come again one per config below
			continue
		}
		name := strings.TrimSpace(text(xray["remarks"]))
		for _, item := range list(xray["outbounds"]) {
			raw := object(item)
			outbound, ok := convertXHTTP(raw)
			if !ok {
				continue
			}
			if name == "" {
				name = text(outbound["server"])
			}
			tag := unique(used, name)
			outbound["tag"] = tag
			proxies = append(proxies, outbound)
			if len(cdnServers) > 0 && !cdnServers[cdnMemberKey(raw)] {
				manualTags = append(manualTags, tag)
				break
			}
			cdnTags = append(cdnTags, tag)
			break
		}
	}

	autoMembers := []string{TagDirect}
	selectorMembers := append([]string{TagAuto}, directTags...)
	selectorMembers = append(selectorMembers, cdnTags...)
	selectorMembers = append(selectorMembers, manualTags...)
	groups := []any{
		map[string]any{"type": "selector", "tag": TagSelector, "outbounds": selectorMembers, "default": TagAuto, "interrupt_exist_connections": true},
		nil,
		urlTest(TagDirect, directTags),
	}
	if len(cdnTags) > 0 {
		autoMembers = append(autoMembers, TagCDN)
		groups = append(groups, urlTest(TagCDN, cdnTags))
	}
	groups[1] = map[string]any{"type": "fallback", "tag": TagAuto, "outbounds": autoMembers, "blacklist_timeout": blacklistTimeout}

	outbounds := append(append(groups, proxies...), others...)
	config["outbounds"] = outbounds
	setPingTestTimeout(config)
	route := object(config["route"])
	if route == nil {
		route = map[string]any{}
		config["route"] = route
	}
	route["final"] = TagSelector

	content, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// setPingTestTimeout keeps the monitor's test limit off the 5 s default; a value the profile sets
// itself wins.
func setPingTestTimeout(config map[string]any) {
	experimental := object(config["experimental"])
	if experimental == nil {
		experimental = map[string]any{}
		config["experimental"] = experimental
	}
	monitoring := object(experimental["monitoring"])
	if monitoring == nil {
		monitoring = map[string]any{}
		experimental["monitoring"] = monitoring
	}
	if _, loaded := monitoring["url_test_timeout"]; !loaded {
		monitoring["url_test_timeout"] = pingTestTimeout
	}
}

func urlTest(tag string, members []string) map[string]any {
	return map[string]any{"type": "urltest", "tag": tag, "outbounds": members, "url": testURL, "interval": testInterval, "tolerance": 50}
}

func isAutopilot(tag string) bool {
	return strings.Contains(strings.ToLower(tag), "автопилот")
}

func unique(used map[string]bool, tag string) string {
	if tag == "" {
		tag = "server"
	}
	result := tag
	for i := 2; used[result]; i++ {
		result = tag + " " + strconv.Itoa(i)
	}
	used[result] = true
	return result
}

// panelAutopilotMembers reads the panel's own autopilot config (the Xray one with balancers):
// "auto*" outbounds are the direct servers, "cdn*" and "fb" are the CDN ones.
func panelAutopilotMembers(configs []map[string]any) (direct map[string]bool, cdn map[string]bool) {
	for _, cfg := range configs {
		if len(list(object(cfg["routing"])["balancers"])) == 0 {
			continue
		}
		direct, cdn = map[string]bool{}, map[string]bool{}
		for _, item := range list(cfg["outbounds"]) {
			outbound := object(item)
			if outbound == nil {
				continue
			}
			tag := text(outbound["tag"])
			switch {
			case strings.HasPrefix(tag, "auto"):
				if address := xrayAddress(outbound); address != "" {
					direct[address] = true
				}
			case strings.HasPrefix(tag, "cdn") || strings.HasPrefix(tag, "fb"):
				if key := cdnMemberKey(outbound); key != "" {
					cdn[key] = true
				}
			}
		}
		return direct, cdn
	}
	return nil, nil
}

func xrayAddress(outbound map[string]any) string {
	settings := object(outbound["settings"])
	for _, item := range list(settings["vnext"]) {
		if address := text(object(item)["address"]); address != "" {
			return address
		}
	}
	if address := text(settings["address"]); address != "" {
		return address
	}
	for _, item := range list(settings["servers"]) {
		if address := text(object(item)["address"]); address != "" {
			return address
		}
	}
	return ""
}

// cdnMemberKey tells CDN locations apart: one front address can carry several locations,
// so the path and the SNI are part of the key.
func cdnMemberKey(outbound map[string]any) string {
	address := xrayAddress(outbound)
	if address == "" {
		return ""
	}
	stream := object(outbound["streamSettings"])
	path := text(object(stream["xhttpSettings"])["path"])
	if path == "" {
		path = text(object(stream["splithttpSettings"])["path"])
	}
	return address + "|" + path + "|" + text(object(stream["tlsSettings"])["serverName"])
}

// convertXHTTP converts an Xray VLESS XHTTP outbound to a sing-box one.
func convertXHTTP(outbound map[string]any) (map[string]any, bool) {
	if outbound == nil || text(outbound["protocol"]) != "vless" {
		return nil, false
	}
	stream := object(outbound["streamSettings"])
	network := text(stream["network"])
	if network != "xhttp" && network != "splithttp" {
		return nil, false
	}
	settings := object(outbound["settings"])
	var address, id, flow string
	var port int
	if vnext := list(settings["vnext"]); len(vnext) > 0 {
		server := object(vnext[0])
		address, port = text(server["address"]), integer(server["port"])
		if users := list(server["users"]); len(users) > 0 {
			user := object(users[0])
			id, flow = text(user["id"]), text(user["flow"])
		}
	} else {
		address, port = text(settings["address"]), integer(settings["port"])
		id, flow = text(settings["id"]), text(settings["flow"])
	}
	if address == "" || id == "" {
		return nil, false
	}
	if port == 0 {
		port = 443
	}
	result := map[string]any{"type": "vless", "server": address, "server_port": port, "uuid": id}
	if flow != "" {
		result["flow"] = flow
	}

	security := text(stream["security"])
	if security == "tls" || security == "reality" {
		key := "tlsSettings"
		if security == "reality" {
			key = "realitySettings"
		}
		tlsSettings := object(stream[key])
		serverName := text(tlsSettings["serverName"])
		if serverName == "" {
			serverName = address
		}
		fingerprint := text(tlsSettings["fingerprint"])
		if fingerprint == "" {
			fingerprint = "chrome"
		}
		tls := map[string]any{"enabled": true, "server_name": serverName, "utls": map[string]any{"enabled": true, "fingerprint": fingerprint}}
		if alpn := list(tlsSettings["alpn"]); len(alpn) > 0 {
			tls["alpn"] = alpn
		}
		if security == "reality" {
			tls["reality"] = map[string]any{"enabled": true, "public_key": text(tlsSettings["publicKey"]), "short_id": text(tlsSettings["shortId"])}
		}
		result["tls"] = tls
	}

	xhttp := object(stream["xhttpSettings"])
	if xhttp == nil {
		xhttp = object(stream["splithttpSettings"])
	}
	extra := map[string]any{}
	for k, v := range object(xhttp["extra"]) {
		extra[k] = v
	}
	for k, v := range xhttp {
		if k != "extra" {
			extra[k] = v
		}
	}
	// Xray reads these names without regard to case (the panel writes SessionIDKey, SessionIDPlacement),
	// so they are looked up in lower case.
	extraLower := make(map[string]any, len(extra))
	for k, v := range extra {
		extraLower[strings.ToLower(k)] = v
	}
	transport := map[string]any{"type": "xhttp", "mode": orDefault(text(xhttp["mode"]), "auto"), "path": orDefault(text(xhttp["path"]), "/")}
	if host := text(xhttp["host"]); host != "" {
		transport["host"] = host
	}
	for xrayKey, singKey := range xhttpKeys {
		if v, loaded := extraLower[strings.ToLower(xrayKey)]; loaded {
			transport[singKey] = valueRange(v)
		}
	}
	if v, loaded := extraLower["uplinkchunksize"]; loaded {
		transport["uplink_chunk_size"] = firstInteger(v)
	}
	headers := map[string]any{}
	for k, v := range object(extraLower["headers"]) {
		if strings.ToLower(k) != "host" {
			headers[k] = v
		}
	}
	if len(headers) > 0 {
		transport["headers"] = headers
	}
	xmux := map[string]any{}
	for k, v := range object(extraLower["xmux"]) {
		if strings.EqualFold(k, "hKeepAlivePeriod") {
			xmux["h_keep_alive_period"] = firstInteger(v)
		} else if singKey, loaded := xmuxKeys[strings.ToLower(k)]; loaded {
			xmux[singKey] = valueRange(v)
		}
	}
	if len(xmux) > 0 {
		transport["xmux"] = xmux
	}
	result["transport"] = transport
	return result, true
}

var xhttpKeys = map[string]string{
	"xPaddingBytes":        "x_padding_bytes",
	"xPaddingObfsMode":     "x_padding_obfs_mode",
	"xPaddingPlacement":    "x_padding_placement",
	"xPaddingKey":          "x_padding_key",
	"xPaddingHeader":       "x_padding_header",
	"xPaddingMethod":       "x_padding_method",
	"noGRPCHeader":         "no_grpc_header",
	"noSSEHeader":          "no_sse_header",
	"scMaxEachPostBytes":   "sc_max_each_post_bytes",
	"scMinPostsIntervalMs": "sc_min_posts_interval_ms",
	"sessionPlacement":     "session_placement",
	"sessionKey":           "session_key",
	"sessionIDPlacement":   "session_placement",
	"sessionIDKey":         "session_key",
	"sessionIDLength":      "session_id_length",
	"sessionIDTable":       "session_id_table",
	"seqPlacement":         "seq_placement",
	"seqKey":               "seq_key",
	"uplinkHTTPMethod":     "uplink_http_method",
	"uplinkDataPlacement":  "uplink_data_placement",
	"uplinkDataKey":        "uplink_data_key",
}

// the keys are lower case: they are compared with the lower-cased Xray names
var xmuxKeys = map[string]string{
	"maxconcurrency":   "max_concurrency",
	"maxconnections":   "max_connections",
	"cmaxreusetimes":   "c_max_reuse_times",
	"hmaxrequesttimes": "h_max_request_times",
	"hmaxreusablesecs": "h_max_reusable_secs",
	// Xray 26 names for the same options
	"maxreusetimes":   "c_max_reuse_times",
	"maxrequesttimes": "h_max_request_times",
	"maxreusablesecs": "h_max_reusable_secs",
}

func object(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func list(v any) []any {
	l, _ := v.([]any)
	return l
}

func text(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
	return ""
}

func integer(v any) int {
	switch value := v.(type) {
	case json.Number:
		i, err := value.Int64()
		if err != nil {
			f, _ := value.Float64()
			return int(f)
		}
		return int(i)
	case float64:
		return int(value)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(value))
		return i
	}
	return 0
}

func firstInteger(v any) int {
	switch value := v.(type) {
	case map[string]any:
		return integer(value["from"])
	case string:
		return integer(strings.Split(value, "-")[0])
	}
	return integer(v)
}

func valueRange(v any) any {
	if m, ok := v.(map[string]any); ok {
		if from, loaded := m["from"]; loaded {
			to, loaded := m["to"]
			if !loaded {
				to = from
			}
			return map[string]any{"from": from, "to": to}
		}
	}
	return v
}

func orDefault(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
