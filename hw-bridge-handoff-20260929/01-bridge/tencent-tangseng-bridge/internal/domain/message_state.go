package domain

// MessageRecord holds native identifiers needed for recall and read receipts.
// Existing delivery mappings remain the source of message deduplication.
type MessageRecord struct {
	TencentSDKID         string `json:"tencent_sdk_id,omitempty"`
	TencentSeq           int64  `json:"tencent_seq,omitempty"`
	Abstract             string `json:"abstract,omitempty"`
	NeedReceipt          bool   `json:"need_receipt"`
	Key                  string `json:"key"`
	Source               Source `json:"source"`
	TencentKey           string `json:"tencent_key"`
	TencentFrom          string `json:"tencent_from"`
	TencentTo            string `json:"tencent_to"`
	TangID               string `json:"tang_id"`
	TangSeq              int64  `json:"tang_seq"`
	TangClientNo         string `json:"tang_client_no"`
	TangFrom             string `json:"tang_from"`
	TangTo               string `json:"tang_to"`
	ChannelType          int    `json:"channel_type"`
	OccurredAt           int64  `json:"occurred_at"`
	ReadSynced           bool   `json:"read_synced"`
	TencentReceiptSynced bool   `json:"tencent_receipt_synced,omitempty"`
	Revoked              bool   `json:"revoked"`
}

type SyncCursor struct {
	Key       string `json:"key"`
	Value     int64  `json:"value"`
	Aux       string `json:"aux,omitempty"`
	StartedAt int64  `json:"started_at"`
}
