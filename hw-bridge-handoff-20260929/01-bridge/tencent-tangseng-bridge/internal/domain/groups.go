package domain

type GroupMember struct {
	UID        string `json:"uid"`
	Name       string `json:"name"`
	Role       int    `json:"role"`
	Version    int64  `json:"version"`
	Deleted    int    `json:"is_deleted"`
	MutedUntil int64  `json:"forbidden_expir_time"`
	TencentID  string `json:"tencent_id,omitempty"`
	Huawei     bool   `json:"huawei,omitempty"`
}
type GroupState struct {
	Avatar         string        `json:"bridge_avatar,omitempty"`
	Number         string        `json:"group_no"`
	TencentID      string        `json:"tencent_id"`
	Creator        string        `json:"creator"`
	Name           string        `json:"name"`
	Notice         string        `json:"notice"`
	Status         int           `json:"status"`
	Forbidden      int           `json:"forbidden"`
	Invite         int           `json:"invite"`
	GroupType      int           `json:"group_type"`
	Members        []GroupMember `json:"members"`
	RequestOwner   string        `json:"request_owner,omitempty"`
	RequestID      string        `json:"request_id,omitempty"`
	MirrorReady    bool          `json:"mirror_ready"`
	PendingInvites []string      `json:"pending_invites,omitempty"`
}
