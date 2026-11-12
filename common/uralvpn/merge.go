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

	used := map[string]bool{}
	var directTags []string
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
			outbound, ok := convertXHTTP(object(item))
			if !ok {
				continue
			}
			if name == "" {
				name = text(outbound["server"])
			}
			tag := unique(used, name)
			outbound["tag"] = tag
			proxies = append(proxies, outbound)
			cdnTags = append(cdnTags, tag)
			break
		}
	}

	autoMembers := []string{TagDirect}
	groups := []any{
		map[string]any{"type": "selector", "tag": TagSelector, "outbounds": append(append([]string{TagAuto}, directTags...), cdnTags...), "default": TagAuto, "interrupt_exist_connections": true},
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
	transport := map[string]any{"type": "xhttp", "mode": orDefault(text(xhttp["mode"]), "auto"), "path": orDefault(text(xhttp["path"]), "/")}
	if host := text(xhttp["host"]); host != "" {
		transport["host"] = host
	}
	for xrayKey, singKey := range xhttpKeys {
		if v, loaded := extra[xrayKey]; loaded {
			transport[singKey] = valueRange(v)
		}
	}
	if v, loaded := extra["uplinkChunkSize"]; loaded {
		transport["uplink_chunk_size"] = firstInteger(v)
	}
	headers := map[string]any{}
	for k, v := range object(extra["headers"]) {
		if strings.ToLower(k) != "host" {
			headers[k] = v
		}
	}
	if len(headers) > 0 {
		transport["headers"] = headers
	}
	xmux := map[string]any{}
	for k, v := range object(extra["xmux"]) {
		if k == "hKeepAlivePeriod" {
			xmux["h_keep_alive_period"] = firstInteger(v)
		} else if singKey, loaded := xmuxKeys[k]; loaded {
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
	"scMaxEachPostBytes":   "sc_max_each_post_bytes",
	"scMinPostsIntervalMs": "sc_min_posts_interval_ms",
	"sessionPlacement":     "session_placement",
	"sessionKey":           "session_key",
	"seqPlacement":         "seq_placement",
	"seqKey":               "seq_key",
	"uplinkHTTPMethod":     "uplink_http_method",
	"uplinkDataPlacement":  "uplink_data_placement",
	"uplinkDataKey":        "uplink_data_key",
}

var xmuxKeys = map[string]string{
	"maxConcurrency":   "max_concurrency",
	"maxConnections":   "max_connections",
	"cMaxReuseTimes":   "c_max_reuse_times",
	"hMaxRequestTimes": "h_max_request_times",
	"hMaxReusableSecs": "h_max_reusable_secs",
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
