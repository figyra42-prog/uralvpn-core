package uralvpn

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// Descriptions returns {"server name": "description"} read from the panel's Xray subscription:
// every config has its name in "remarks" and its text in meta.serverDescription (the field Happ
// shows under the server). The text is plain, "base64:..." or bare base64.
func Descriptions(xrayConfig string) (string, error) {
	var xrayConfigs []map[string]any
	if err := decode(xrayConfig, &xrayConfigs); err != nil {
		var single map[string]any
		if err2 := decode(xrayConfig, &single); err2 != nil {
			return "", err
		}
		xrayConfigs = []map[string]any{single}
	}
	result := map[string]string{}
	for _, xray := range xrayConfigs {
		name := strings.TrimSpace(text(xray["remarks"]))
		if len(list(object(xray["routing"])["balancers"])) > 0 {
			name = TagAuto // the panel's autopilot entry is our autopilot
		}
		raw := text(object(xray["meta"])["serverDescription"])
		if raw == "" {
			raw = text(xray["serverDescription"])
		}
		description := decodeDescription(raw)
		if name != "" && description != "" {
			result[name] = description
		}
	}
	content, err := json.Marshal(result)
	return string(content), err
}

func decodeDescription(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(value), "base64:") {
		value = strings.TrimSpace(value[len("base64:"):])
		if decoded, ok := decodeBase64Text(value); ok {
			return decoded
		}
		return ""
	}
	if !strings.ContainsAny(value, " \n") {
		if decoded, ok := decodeBase64Text(value); ok {
			return decoded
		}
	}
	return limitText(value)
}

func decodeBase64Text(value string) (string, bool) {
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		data, err := encoding.DecodeString(value)
		if err == nil && utf8.Valid(data) && len(data) > 0 {
			return limitText(string(data)), true
		}
	}
	return "", false
}

func limitText(value string) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > 300 {
		value = string([]rune(value)[:300])
	}
	return value
}
