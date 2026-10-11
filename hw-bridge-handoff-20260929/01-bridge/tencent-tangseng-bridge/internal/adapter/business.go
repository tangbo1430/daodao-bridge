package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// UserRequest never falls back to a service token: mutations must run as the actor.
func (c *TangSengClient) UserRequest(ctx context.Context, uid, method, path string, body, result any) error {
	var attemptedToken string
	if c.UserTokens != nil {
		v, _ := c.UserTokens.Load(uid)
		attemptedToken, _ = v.(string)
	}
	err := c.userRequest(ctx, uid, method, path, body, result)
	var status *HTTPStatusError
	if errors.As(err, &status) && status.Status == http.StatusUnauthorized && (c.RefreshUser != nil || c.RefreshUserIfToken != nil) {
		var refreshErr error
		if c.RefreshUserIfToken != nil {
			refreshErr = c.RefreshUserIfToken(ctx, uid, attemptedToken)
		} else {
			refreshErr = c.RefreshUser(ctx, uid)
		}
		if refreshErr != nil {
			return refreshErr
		}
		return nativeFailure(c.userRequest(ctx, uid, method, path, body, result))
	}
	return nativeFailure(err)
}
func (c *TangSengClient) userRequest(ctx context.Context, uid, method, path string, body, result any) error {
	if c == nil || c.BaseURL == "" || c.UserTokens == nil {
		return fmt.Errorf("用户尚未登录")
	}
	value, ok := c.UserTokens.Load(uid)
	token, _ := value.(string)
	if !ok || token == "" {
		return fmt.Errorf("登录已失效，请重新登录")
	}
	var raw json.RawMessage
	if err := c.HTTP.DoJSON(ctx, method, c.endpoint(path), token, body, &raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		return nil
	}
	var status struct {
		Status  int    `json:"status"`
		Code    int    `json:"code"`
		Message string `json:"msg"`
	}
	if json.Unmarshal(raw, &status) == nil && (status.Status >= 400 || (status.Code != 0 && status.Code != 200)) {
		if status.Status == 401 || status.Code == 401 {
			return &HTTPStatusError{401, "user authentication expired"}
		}
		code := status.Status
		if code == 0 {
			code = status.Code
		}
		return &HTTPStatusError{code, status.Message}
	}
	if result != nil {
		return json.Unmarshal(raw, result)
	}
	return nil
}

func (c *TangSengClient) FriendSure(ctx context.Context, owner, token string) error {
	return c.UserRequest(ctx, owner, http.MethodPost, "/v1/friend/sure", map[string]string{"token": token}, nil)
}
func (c *TangSengClient) FriendRefuse(ctx context.Context, owner, peer string) error {
	return c.UserRequest(ctx, owner, http.MethodPut, "/v1/friend/refuse/"+url.PathEscape(peer), nil, nil)
}
func (c *TangSengClient) FriendDelete(ctx context.Context, owner, peer string) error {
	return c.UserRequest(ctx, owner, http.MethodDelete, "/v1/friends/"+url.PathEscape(peer), nil, nil)
}
func (c *TangSengClient) Blacklist(ctx context.Context, owner, peer string, blocked bool) error {
	method := http.MethodDelete
	if blocked {
		method = http.MethodPost
	}
	return c.UserRequest(ctx, owner, method, "/v1/user/blacklist/"+url.PathEscape(peer), nil, nil)
}

type TencentResult struct {
	ActionStatus string `json:"ActionStatus"`
	ErrorCode    int    `json:"ErrorCode"`
	ErrorInfo    string `json:"ErrorInfo"`
	ResultItem   []struct {
		ToAccount string `json:"To_Account"`
		Code      int    `json:"ResultCode"`
		Info      string `json:"ResultInfo"`
	} `json:"ResultItem"`
}
type TencentError struct {
	Code      int
	Operation string
}

func (e *TencentError) Error() string {
	return fmt.Sprintf("Tencent %s failed: %d", e.Operation, e.Code)
}
func (c TencentClient) BusinessRequest(ctx context.Context, service, command string, body any, result any) error {
	var raw json.RawMessage
	if err := c.request(ctx, service, command, body, &raw); err != nil {
		return err
	}
	var response TencentResult
	if err := json.Unmarshal(raw, &response); err != nil {
		return err
	}
	if response.ErrorCode != 0 || response.ActionStatus != "OK" {
		return fmt.Errorf("Tencent %s failed: %d %s", command, response.ErrorCode, response.ErrorInfo)
	}
	if result != nil {
		return json.Unmarshal(raw, result)
	}
	return nil
}
func (c TencentClient) FriendAdd(ctx context.Context, from, to, wording string, force bool) error {
	return c.friendAddType(ctx, from, to, wording, force, "Add_Type_Both")
}
func (c TencentClient) friendAddType(ctx context.Context, from, to, wording string, force bool, kind string) error {
	flags := 0
	if force {
		flags = 1
	}
	var result TencentResult
	err := c.request(ctx, "sns", "friend_add", map[string]any{"From_Account": from, "AddFriendItem": []any{map[string]any{"To_Account": to, "AddSource": "AddSource_Type_TSDD", "AddWording": wording}}, "AddType": kind, "ForceAddFlags": flags}, &result)
	if err != nil {
		return err
	}
	if len(result.ResultItem) == 0 {
		if result.ErrorCode != 0 {
			return &TencentError{result.ErrorCode, "friend_add"}
		}
		if result.ActionStatus != "OK" {
			return fmt.Errorf("Tencent friend_add returned no result")
		}
	}
	for _, item := range result.ResultItem {
		if item.Code != 0 && item.Code != 30015 && !(item.Code == 30539 && !force) {
			return &TencentError{item.Code, "friend_add"}
		}
	}
	return nil
}
func (c TencentClient) DeleteFriend(ctx context.Context, from, to string) error {
	return c.deleteFriendType(ctx, from, to, "Delete_Type_Both")
}
func (c TencentClient) SetFriendEdge(ctx context.Context, from, to string, present bool) error {
	if present {
		return c.friendAddType(ctx, from, to, "", true, "Add_Type_Single")
	}
	return c.deleteFriendType(ctx, from, to, "Delete_Type_Single")
}
func (c TencentClient) deleteFriendType(ctx context.Context, from, to, kind string) error {
	var result TencentResult
	if err := c.BusinessRequest(ctx, "sns", "friend_delete", map[string]any{"From_Account": from, "To_Account": []string{to}, "DeleteType": kind}, &result); err != nil {
		return err
	}
	for _, item := range result.ResultItem {
		if item.Code == 30003 && item.ToAccount == to {
			absent, err := c.friendDeleteTargetAbsent(ctx, from, to)
			if err != nil {
				return err
			}
			if absent {
				continue
			}
		}
		if item.Code != 0 && item.Code != 31704 {
			return fmt.Errorf("Tencent friend_delete item failed: %d", item.Code)
		}
	}
	return nil
}

// A deleted Tang relation can refer to a shadow never imported into Tencent.
// Only a verified missing target is an already-completed delete; a missing
// owner, invalid response or permission error must remain a failure.
func (c TencentClient) friendDeleteTargetAbsent(ctx context.Context, from, to string) (bool, error) {
	var result struct {
		Items []struct {
			UserID   string `json:"UserID"`
			LegacyID string `json:"Identifier"`
			Code     int    `json:"ResultCode"`
			Status   string `json:"AccountStatus"`
		} `json:"ResultItem"`
	}
	if err := c.BusinessRequest(ctx, "im_open_login_svc", "account_check", map[string]any{"CheckItem": []any{map[string]string{"UserID": from}, map[string]string{"UserID": to}}}, &result); err != nil {
		return false, err
	}
	statuses := map[string]string{}
	for _, item := range result.Items {
		uid := item.UserID
		if uid == "" {
			uid = item.LegacyID
		}
		if item.Code != 0 {
			return false, &TencentError{item.Code, "account_check"}
		}
		statuses[uid] = item.Status
	}
	return statuses[from] == "Imported" && statuses[to] == "NotImported", nil
}
func (c TencentClient) UpdateProfile(ctx context.Context, uid, name, avatar string) error {
	items := []any{map[string]any{"Tag": "Tag_Profile_IM_Nick", "Value": name}}
	if strings.TrimSpace(avatar) != "" {
		items = append(items, map[string]any{"Tag": "Tag_Profile_IM_Image", "Value": avatar})
	}
	return c.BusinessRequest(ctx, "profile", "portrait_set", map[string]any{"From_Account": uid, "ProfileItem": items}, nil)
}

// FriendAllowType is read without changing the account's native Tencent policy.
func (c TencentClient) FriendAllowType(ctx context.Context, uid string) (string, error) {
	var result struct {
		Users []struct {
			Account string `json:"To_Account"`
			Code    int    `json:"ResultCode"`
			Items   []struct {
				Tag   string `json:"Tag"`
				Value string `json:"Value"`
			} `json:"ProfileItem"`
		} `json:"UserProfileItem"`
	}
	if err := c.BusinessRequest(ctx, "profile", "portrait_get", map[string]any{"To_Account": []string{uid}, "TagList": []string{"Tag_Profile_IM_AllowType"}}, &result); err != nil {
		return "", err
	}
	for _, user := range result.Users {
		if user.Account == uid {
			if user.Code != 0 {
				return "", &TencentError{user.Code, "portrait_get"}
			}
			for _, item := range user.Items {
				if item.Tag == "Tag_Profile_IM_AllowType" {
					return item.Value, nil
				}
			}
		}
	}
	return "", fmt.Errorf("Tencent friend policy missing")
}

func (c TencentClient) SendFriendApplication(ctx context.Context, from, to, requestID, name, wording string) error {
	data, err := json.Marshal(map[string]any{"businessID": "bridge_friend_apply", "user_id": from, "request_id": requestID, "content": name + " 申请添加你为好友：" + wording + "（点击处理）"})
	if err != nil {
		return err
	}
	marker, _ := json.Marshal(map[string]string{"bridge_source": "tangseng", "bridge_event_id": requestID})
	return c.BusinessRequest(ctx, "openim", "sendmsg", map[string]any{
		"From_Account": from, "To_Account": to, "SyncOtherMachine": 1,
		"MsgRandom": stableMessageRandom("friend_apply", from, to, requestID), "CloudCustomData": string(marker),
		"MsgBody": []any{map[string]any{"MsgType": "TIMCustomElem", "MsgContent": map[string]any{"Data": string(data), "Desc": "好友申请", "Ext": "bridge_friend_apply"}}},
	}, nil)
}
