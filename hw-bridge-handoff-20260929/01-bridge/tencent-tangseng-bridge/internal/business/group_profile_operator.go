package business

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
)

// Attribution is scoped to fields changed by a verified action. A sync reader
// must never be reported as the author of unrelated pending profile changes.
type groupProfileChange struct {
	NativeOperator string
	Values         map[string]string
	Avatar         bool
}

func newGroupProfileChange(uid, field, value string) groupProfileChange {
	c := groupProfileChange{NativeOperator: uid, Values: map[string]string{}}
	switch field {
	case "name":
		c.Values["Name"] = truncateUTF8(value, 30)
	case "notice":
		c.Values["Notification"] = truncateUTF8(value, 300)
	case "forbidden":
		if value == "0" {
			c.Values["MuteAllMember"] = "Off"
		}
		if value == "1" {
			c.Values["MuteAllMember"] = "On"
		}
	case "avatar":
		c.Avatar = true // successful authenticated upload; URL is rehosted
	}
	return c
}

func profileEchoKey(group, operator, field, value string) string {
	return "group-profile-echo:" + domain.StableID(group, operator, field, value)
}

// Persist before create, so a delayed avatar or a lost create response cannot
// substitute the PC(2) observer for the group creation identity on retry.
func (s *Service) rememberGroupAvatarInitializer(ctx context.Context, g domain.GroupState, owner string) error {
	key := "group-avatar-initializer:" + g.Number
	if _, err := s.Store.GetCursor(ctx, key); err == nil {
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	for _, m := range g.Members {
		if m.TencentID == owner && m.Deleted == 0 {
			return s.Store.PutCursor(ctx, domain.SyncCursor{Key: key, Aux: m.UID, Value: time.Now().Unix()})
		}
	}
	return nil
}

func (s *Service) projectGroupProfile(ctx context.Context, g domain.GroupState, profile map[string]any, change groupProfileChange) error {
	operator := ""
	for _, member := range g.Members {
		if member.UID == change.NativeOperator && member.Deleted == 0 {
			operator = member.TencentID
			break
		}
	}
	attributed := map[string]any{"GroupId": g.TencentID}
	background := map[string]any{"GroupId": g.TencentID}
	for field, value := range profile {
		if field == "GroupId" {
			continue
		}
		want, specified := change.Values[field]
		if operator != "" && ((specified && value == want) || (field == "FaceUrl" && change.Avatar)) {
			attributed[field] = value
		} else {
			background[field] = value
		}
	}
	if len(attributed) > 1 {
		// Persist before sending: a callback can arrive before the REST response,
		// or the response may be lost even though Tencent applied the change.
		for field, value := range attributed {
			if field == "GroupId" {
				continue
			}
			if err := s.Store.PutCursor(ctx, domain.SyncCursor{Key: profileEchoKey(g.TencentID, operator, field, value.(string)), Value: time.Now().UnixMilli()}); err != nil {
				return err
			}
		}
		attributed["From_Account"] = operator
		if err := s.Tencent.GroupRequest(ctx, "modify_group_base_info", attributed, nil); err != nil {
			return err
		}
	}
	if len(background) > 1 {
		return s.Tencent.GroupRequest(ctx, "modify_group_base_info", background, nil)
	}
	return nil
}

// Match the precise mirrored field before looking up a Huawei login. A native
// shadow operator has no Huawei login and must not cause a reverse write.
func (s *Service) isProfileEcho(ctx context.Context, group, operator, field, value string, eventTime int64) (bool, error) {
	c, err := s.Store.GetCursor(ctx, profileEchoKey(group, operator, field, value))
	if errors.Is(err, store.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if eventTime == 0 {
		eventTime = time.Now().UnixMilli()
	}
	// Tencent's documented EventTime is milliseconds. Keep delayed retries
	// recognisable using their original event time, not worker wall time.
	return eventTime >= c.Value-5000 && eventTime <= c.Value+int64(10*time.Minute/time.Millisecond), nil
}

func (s *Service) filterProfileEcho(ctx context.Context, e domain.BusinessEvent) (domain.BusinessEvent, bool, error) {
	if e.Command != "Group.CallbackAfterGroupInfoChanged" {
		return e, false, nil
	}
	var p struct {
		Group     string `json:"GroupId"`
		Operator  string `json:"Operator_Account"`
		EventTime int64  `json:"EventTime"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return e, false, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(e.Payload, &fields); err != nil {
		return e, false, err
	}
	remaining := false
	for _, field := range []string{"Name", "Notification", "FaceUrl"} {
		var value string
		if json.Unmarshal(fields[field], &value) != nil {
			continue
		}
		match, err := s.isProfileEcho(ctx, p.Group, p.Operator, field, value, p.EventTime)
		if err != nil {
			return e, false, err
		}
		if match {
			delete(fields, field)
		} else {
			remaining = true
		}
	}
	if !remaining {
		return e, true, nil
	}
	var err error
	e.Payload, err = json.Marshal(fields)
	return e, false, err
}
