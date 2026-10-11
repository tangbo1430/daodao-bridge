package session

import "encoding/json"

// JSON-RPC uses an object; HTTP history uses the public WuKong bit field.
func receiptSetting(raw json.RawMessage) bool {
	var bits uint8
	if json.Unmarshal(raw, &bits) == nil {
		return bits&128 != 0
	}
	var v struct {
		Receipt json.RawMessage `json:"receipt"`
	}
	if json.Unmarshal(raw, &v) != nil {
		return false
	}
	var enabled bool
	if json.Unmarshal(v.Receipt, &enabled) == nil {
		return enabled
	}
	var n int
	return json.Unmarshal(v.Receipt, &n) == nil && n == 1
}
