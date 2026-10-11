package adapter

import (
	"context"
	"fmt"
)

type TencentGroupProfile struct {
	ID           string `json:"GroupId"`
	Code         int    `json:"ErrorCode"`
	Max          int    `json:"MaxMemberNum"`
	Owner        string `json:"Owner_Account"`
	Name         string `json:"Name"`
	Introduction string `json:"Introduction"`
	Notification string `json:"Notification"`
	FaceURL      string `json:"FaceUrl"`
	MuteAll      string `json:"MuteAllMember"`
}

type TencentGroupMember struct {
	ID          string `json:"Member_Account"`
	Role        string `json:"Role"`
	ShutUpUntil int64  `json:"ShutUpUntil"`
	MuteUntil   int64  `json:"MuteUntil"`
}

func (m TencentGroupMember) MutedUntil() int64 {
	if m.MuteUntil > m.ShutUpUntil {
		return m.MuteUntil
	}
	return m.ShutUpUntil
}

func (c TencentClient) GroupProfile(ctx context.Context, group string) (TencentGroupProfile, error) {
	var result struct {
		Groups []TencentGroupProfile `json:"GroupInfo"`
	}
	if err := c.GroupRequest(ctx, "get_group_info", map[string]any{"GroupIdList": []string{group}, "ResponseFilter": map[string]any{"GroupBaseInfoFilter": []string{"MaxMemberNum", "Owner_Account", "Name", "Introduction", "Notification", "FaceUrl", "MuteAllMember"}}}, &result); err != nil {
		return TencentGroupProfile{}, err
	}
	if len(result.Groups) != 1 || result.Groups[0].ID != group || result.Groups[0].Code != 0 || result.Groups[0].Max < 1 {
		return TencentGroupProfile{}, fmt.Errorf("Tencent group profile unavailable")
	}
	return result.Groups[0], nil
}

func (c TencentClient) GroupRequest(ctx context.Context, command string, body any, result any) error {
	return c.BusinessRequest(ctx, "group_open_http_svc", command, body, result)
}

func (c TencentClient) DestroyMirrorGroup(ctx context.Context, group string) error {
	var result TencentResult
	if err := c.request(ctx, "group_open_http_svc", "destroy_group", map[string]string{"GroupId": group}, &result); err != nil {
		return err
	}
	if result.ErrorCode == 10010 {
		return nil
	} // Already absent after a previous successful destroy.
	if result.ErrorCode != 0 || result.ActionStatus != "OK" {
		return &TencentError{result.ErrorCode, "destroy_group"}
	}
	return nil
}

func (c TencentClient) GroupMembers(ctx context.Context, group string) ([]string, error) {
	members, err := c.GroupMemberProfiles(ctx, group)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, m := range members {
		out = append(out, m.ID)
	}
	return out, nil
}

func (c TencentClient) GroupMemberProfiles(ctx context.Context, group string) ([]TencentGroupMember, error) {
	out := []TencentGroupMember{}
	for offset := 0; offset < 100000; offset += 100 {
		var result struct {
			Next    int                  `json:"Next"`
			Members []TencentGroupMember `json:"MemberList"`
		}
		if err := c.GroupRequest(ctx, "get_group_member_info", map[string]any{"GroupId": group, "Limit": 100, "Offset": offset}, &result); err != nil {
			return nil, err
		}
		for _, m := range result.Members {
			out = append(out, m)
		}
		if len(result.Members) < 100 {
			return out, nil
		}
	}
	return nil, fmt.Errorf("Tencent group member page limit exceeded")
}
func (c TencentClient) CreateMirrorGroup(ctx context.Context, body map[string]any) error {
	var result TencentResult
	if err := c.request(ctx, "group_open_http_svc", "create_group", body, &result); err != nil {
		return err
	}
	if result.ErrorCode == 10021 || result.ErrorCode == 10025 {
		return nil
	}
	if result.ErrorCode != 0 || result.ActionStatus != "OK" {
		return &TencentError{result.ErrorCode, "create_group"}
	}
	return nil
}
func (c TencentClient) AddGroupMembers(ctx context.Context, group string, members []string) error {
	for start := 0; start < len(members); start += 100 {
		end := start + 100
		if end > len(members) {
			end = len(members)
		}
		items := make([]map[string]string, 0, end-start)
		for _, uid := range members[start:end] {
			items = append(items, map[string]string{"Member_Account": uid})
		}
		var result struct {
			Members []struct {
				Account string `json:"Member_Account"`
				Result  int    `json:"Result"`
			} `json:"MemberList"`
		}
		if err := c.GroupRequest(ctx, "add_group_member", map[string]any{"GroupId": group, "Silence": 1, "MemberList": items}, &result); err != nil {
			return err
		}
		if len(result.Members) != len(items) {
			return fmt.Errorf("Tencent group member result incomplete")
		}
		for _, m := range result.Members {
			if m.Result != 1 && m.Result != 2 {
				return fmt.Errorf("Tencent group member rejected: result=%d", m.Result)
			}
		}
	}
	return nil
}
func (c TencentClient) DeleteGroupMembers(ctx context.Context, group string, members []string) error {
	for start := 0; start < len(members); start += 100 {
		end := start + 100
		if end > len(members) {
			end = len(members)
		}
		if err := c.GroupRequest(ctx, "delete_group_member", map[string]any{"GroupId": group, "Silence": 1, "MemberToDel_Account": members[start:end]}, nil); err != nil {
			return err
		}
	}
	return nil
}
