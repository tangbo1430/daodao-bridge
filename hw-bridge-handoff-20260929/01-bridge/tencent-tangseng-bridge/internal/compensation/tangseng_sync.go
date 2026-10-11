package compensation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
)

// TangSyncClient implements the incremental F pull contract. Cursor values are
// opaque to the bridge and are advanced only by the caller after inbox
// acceptance, so a replay cannot skip a message.
type TangSyncClient struct {
	BaseURL, Token string
	HTTP           adapter.HTTPClient
}

type SyncRequest struct {
	UID, Cursor string
	Limit       int
}
type SyncMessage struct {
	MessageID   string `json:"message_id"`
	MessageSeq  int64  `json:"message_seq"`
	ChannelID   string `json:"channel_id"`
	ChannelType int    `json:"channel_type"`
	FromUID     string `json:"from_uid"`
	Payload     any    `json:"payload"`
}
type SyncResponse struct {
	Messages   []SyncMessage `json:"messages"`
	NextCursor string        `json:"next_cursor"`
	HasMore    bool          `json:"has_more"`
}

func (c TangSyncClient) endpoint(path string) string {
	return strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
}
func (c TangSyncClient) Pull(ctx context.Context, req SyncRequest) (SyncResponse, error) {
	if c.BaseURL == "" {
		return SyncResponse{}, fmt.Errorf("TangSeng sync base URL is not configured")
	}
	if req.Limit <= 0 || req.Limit > 500 {
		req.Limit = 100
	}
	body := map[string]any{"max_message_seq": req.Cursor, "limit": req.Limit}
	var raw json.RawMessage
	if err := c.HTTP.DoJSON(ctx, http.MethodPost, c.endpoint("/v1/message/sync"), c.Token, body, &raw); err != nil {
		return SyncResponse{}, err
	}
	var response SyncResponse
	if len(raw) > 0 && raw[0] == '[' {
		if err := json.Unmarshal(raw, &response.Messages); err != nil {
			return SyncResponse{}, err
		}
	} else if err := json.Unmarshal(raw, &response); err != nil {
		return SyncResponse{}, err
	}
	return response, nil
}
