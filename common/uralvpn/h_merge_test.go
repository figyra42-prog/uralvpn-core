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
