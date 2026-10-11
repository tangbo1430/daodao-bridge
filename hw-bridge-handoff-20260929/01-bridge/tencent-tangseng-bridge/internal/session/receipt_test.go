package session

import (
	"encoding/json"
	"testing"
)

func TestReceiptSetting(t *testing.T) {
	for raw, want := range map[string]bool{`{"receipt":true}`: true, `{"receipt":1}`: true, `128`: true, `160`: true, `0`: false, `32`: false, `{"receipt":false}`: false, `null`: false, `{}`: false} {
		if got := receiptSetting(json.RawMessage(raw)); got != want {
			t.Errorf("%s got %v want %v", raw, got, want)
		}
	}
}
