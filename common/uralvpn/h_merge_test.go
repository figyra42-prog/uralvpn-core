package uralvpn

import (
	"encoding/json"
	"testing"
)

// h_merge_test.go: the autopilot must follow the panel's own lists (the Xray config with
// balancers): a location without the DIRECT tag stays a manual one, and a CDN host the panel
// does not list stays out of the CDN group.
func TestH_MergeAutopilotFollowsPanelLists(t *testing.T) {
	singbox := `{
  "inbounds": [{"type": "mixed", "listen": "127.0.0.1", "listen_port": 2080}],
  "outbounds": [
    {"type": "selector", "tag": "→ Remnawave", "outbounds": ["🇩🇪 Франкфурт"]},
    {"type": "direct", "tag": "direct"},
    {"type": "block", "tag": "block"},
    {"type": "vless", "tag": "🇪🇺  💫 АВТОПИЛОТ🚀", "server": "de.example.com", "server_port": 443, "uuid": "u"},
    {"type": "vless", "tag": "🇩🇪 Франкфурт", "server": "de.example.com", "server_port": 443, "uuid": "u"},
    {"type": "vless", "tag": "🇱🇻 Латвия", "server": "lv.example.com", "server_port": 443, "uuid": "u"},
    {"type": "vless", "tag": "🇷🇺 Россия", "server": "ru.example.com", "server_port": 443, "uuid": "u"},
    {"type": "vless", "tag": "🇳🇱 Нидерланды Hysteria", "server": "nl.example.com", "server_port": 443, "uuid": "u"}
  ]
}`
	xray := `[
  {"routing": {"balancers": [{"tag": "bal", "selector": ["auto"], "strategy": {"type": "leastLoad"}}]},
   "outbounds": [
    {"tag": "fb", "protocol": "vless", "settings": {"vnext": [{"address": "188.72.110.5", "port": 443}]}, "streamSettings": {"network": "xhttp", "xhttpSettings": {"path": "/p"}, "tlsSettings": {"serverName": "s.example.com"}}},
    {"tag": "auto", "protocol": "vless", "settings": {"vnext": [{"address": "de.example.com", "port": 443}]}},
    {"tag": "auto-2", "protocol": "vless", "settings": {"vnext": [{"address": "lv.example.com", "port": 443}]}},
    {"tag": "cdn", "protocol": "vless", "settings": {"vnext": [{"address": "cdn.example.com", "port": 443}]}, "streamSettings": {"network": "xhttp", "xhttpSettings": {"path": "/p2"}, "tlsSettings": {"serverName": "s2.example.com"}}}
   ]},
  {"remarks": "🇩🇪 Германия моб", "outbounds": [{"tag": "p", "protocol": "vless", "settings": {"vnext": [{"address": "188.72.110.5", "port": 443, "users": [{"id": "u"}]}]}, "streamSettings": {"network": "xhttp", "xhttpSettings": {"path": "/p"}, "tlsSettings": {"serverName": "s.example.com"}}}]},
  {"remarks": "🇪🇪 Эстония моб", "outbounds": [{"tag": "p", "protocol": "vless", "settings": {"vnext": [{"address": "cdn.example.com", "port": 443, "users": [{"id": "u"}]}]}, "streamSettings": {"network": "xhttp", "xhttpSettings": {"path": "/p2"}, "tlsSettings": {"serverName": "s2.example.com"}}}]},
  {"remarks": "🇨🇭 Без тега", "outbounds": [{"tag": "p", "protocol": "vless", "settings": {"vnext": [{"address": "x.example.com", "port": 443, "users": [{"id": "u"}]}]}, "streamSettings": {"network": "xhttp", "xhttpSettings": {"path": "/p3"}, "tlsSettings": {"serverName": "s3.example.com"}}}]}
]`
	merged, err := Merge(singbox, xray)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(merged), &cfg); err != nil {
		t.Fatalf("merge output is not JSON: %v", err)
	}
	groups := map[string][]string{}
	for _, item := range cfg["outbounds"].([]any) {
		outbound, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tag, _ := outbound["tag"].(string)
		members, ok := outbound["outbounds"].([]any)
		if !ok {
			continue
		}
		var names []string
		for _, member := range members {
			if name, ok := member.(string); ok {
				names = append(names, name)
			}
		}
		groups[tag] = names
	}
	contains := func(names []string, want string) bool {
		for _, value := range names {
			if value == want {
				return true
			}
		}
		return false
	}
	auto := groups[TagAuto]
	if !contains(auto, TagDirect) || !contains(auto, TagCDN) {
		t.Fatalf("%s must fall back to %s and %s, got %v", TagAuto, TagDirect, TagCDN, auto)
	}
	direct := groups[TagDirect]
	if !contains(direct, "🇩🇪 Франкфурт") || !contains(direct, "🇱🇻 Латвия") {
		t.Errorf("direct servers from the panel's list are missing: %v", direct)
	}
	if contains(direct, "🇷🇺 Россия") || contains(direct, "🇳🇱 Нидерланды Hysteria") {
		t.Errorf("servers without the DIRECT tag got into the autopilot: %v", direct)
	}
	cdn := groups[TagCDN]
	if !contains(cdn, "🇩🇪 Германия моб") || !contains(cdn, "🇪🇪 Эстония моб") {
		t.Errorf("CDN servers from the panel's list are missing: %v", cdn)
	}
	if contains(cdn, "🇨🇭 Без тега") {
		t.Errorf("a CDN server without the CDN tag got into the CDN group: %v", cdn)
	}
	selector := groups[TagSelector]
	for _, want := range []string{"🇷🇺 Россия", "🇳🇱 Нидерланды Hysteria", "🇨🇭 Без тега"} {
		if !contains(selector, want) {
			t.Errorf("manual location %q disappeared from %s: %v", want, TagSelector, selector)
		}
	}
}

// A slow mobile network must not turn a working server into "no data": the monitor's own test
// limit is raised off the 5 s default, and a value the profile sets itself is left alone.
func TestH_MergePingTestTimeout(t *testing.T) {
	singbox := `{"outbounds": [
  {"type": "selector", "tag": "→ Remnawave", "outbounds": ["🇱🇻 Латвия"]},
  {"type": "direct", "tag": "direct"},
  {"type": "vless", "tag": "🇱🇻 Латвия", "server": "lv.example.com", "server_port": 443, "uuid": "u"}
]}`
	merged, err := Merge(singbox, `[]`)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(merged), &cfg); err != nil {
		t.Fatalf("merge output is not JSON: %v", err)
	}
	monitoring, _ := cfg["experimental"].(map[string]any)["monitoring"].(map[string]any)
	if got := monitoring["url_test_timeout"]; got != pingTestTimeout {
		t.Fatalf("url_test_timeout = %v, want %s", got, pingTestTimeout)
	}

	own := `{"experimental": {"monitoring": {"url_test_timeout": "3s"}}, "outbounds": [
  {"type": "selector", "tag": "→ Remnawave", "outbounds": ["🇱🇻 Латвия"]},
  {"type": "direct", "tag": "direct"},
  {"type": "vless", "tag": "🇱🇻 Латвия", "server": "lv.example.com", "server_port": 443, "uuid": "u"}
]}`
	merged, err = Merge(own, `[]`)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	cfg = nil
	if err := json.Unmarshal([]byte(merged), &cfg); err != nil {
		t.Fatalf("merge output is not JSON: %v", err)
	}
	monitoring, _ = cfg["experimental"].(map[string]any)["monitoring"].(map[string]any)
	if got := monitoring["url_test_timeout"]; got != "3s" {
		t.Fatalf("the profile's own url_test_timeout was overwritten: %v", got)
	}
}

// Xray 26 renamed the XMUX options and added noSSEHeader; the panel sends these names in its
// Xray JSON subscription, so the converter must carry them into the sing-box names.
func TestH_ConvertXHTTPXray26Names(t *testing.T) {
	raw := `{
 "tag": "cdn-1", "protocol": "vless",
 "settings": {"vnext": [{"address": "fin.example.com", "port": 443, "users": [{"id": "5db6db0f-54d4-47b0-9360-4d84088e94d2", "encryption": "none"}]}]},
 "streamSettings": {
  "network": "xhttp",
  "xhttpSettings": {
   "mode": "packet-up", "path": "/api/v2/sync/stream/index.json",
   "extra": {
    "noSSEHeader": true, "enableXmux": true,
    "xmux": {"maxReuseTimes": 1, "maxRequestTimes": "0", "maxReusableSecs": "0", "maxConcurrency": "1"},
    "xPaddingKey": "_token", "xPaddingMethod": "repeat-x", "xPaddingBytes": "100-1000",
    "sessionIDKey": "session", "seqKey": "part", "scMaxEachPostBytes": "262144",
    "uplinkDataPlacement": "body", "uplinkHTTPMethod": "GET"
   }
  },
  "tlsSettings": {"serverName": "fin.example.com", "alpn": ["h2"]}
 }
}`
	var outbound map[string]any
	if err := json.Unmarshal([]byte(raw), &outbound); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got, ok := convertXHTTP(outbound)
	if !ok {
		t.Fatal("convertXHTTP: outbound skipped")
	}
	transport, _ := got["transport"].(map[string]any)
	if transport["no_sse_header"] != true {
		t.Fatalf("no_sse_header not passed: %#v", transport["no_sse_header"])
	}
	if transport["x_padding_key"] != "_token" || transport["x_padding_method"] != "repeat-x" {
		t.Fatalf("padding lost: %#v %#v", transport["x_padding_key"], transport["x_padding_method"])
	}
	xmux, _ := transport["xmux"].(map[string]any)
	for _, key := range []string{"c_max_reuse_times", "h_max_request_times", "h_max_reusable_secs"} {
		if _, loaded := xmux[key]; !loaded {
			t.Fatalf("xmux %s not passed: %#v", key, xmux)
		}
	}
}
