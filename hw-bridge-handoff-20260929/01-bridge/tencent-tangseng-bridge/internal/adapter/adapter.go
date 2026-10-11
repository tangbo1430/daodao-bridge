package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/proxy"
	"github.com/tencentyun/tls-sig-api-v2-golang/tencentyun"
)

type Target interface {
	EnsureUser(context.Context, domain.Identity) (domain.Identity, error)
	EnsureGroup(context.Context, domain.Identity) (domain.Identity, error)
	SendText(context.Context, domain.MessageEvent, domain.Identity, domain.Identity, string) (string, error)
}

type HTTPClient struct{ Client *http.Client }

type HTTPStatusError struct {
	Status  int
	Message string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("remote HTTP %d: %s", e.Status, e.Message)
}

func (c HTTPClient) DoJSON(ctx context.Context, method, url, token string, requestBody any, responseBody any) error {
	var bodyReader io.Reader
	if requestBody != nil && method != http.MethodGet && method != http.MethodHead {
		body, err := json.Marshal(requestBody)
		if err != nil {
			return err
		}
		bodyReader = strings.NewReader(string(body))
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		// TangSeng's wkhttp middleware reads the literal `token` header.
		req.Header.Set("token", token)
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyText, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &HTTPStatusError{resp.StatusCode, strings.TrimSpace(string(bodyText))}
	}
	if responseBody == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil {
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("decode remote response: %w", err)
	}
	return nil
}

type TencentClient struct {
	PushAndroidInfo map[string]any
	SDKAppID        int
	SecretKey       string
	AdminAccount    string
	BaseURL         string
	SigTTL          int
	HTTP            HTTPClient
}

func (c TencentClient) authQuery() (string, error) {
	if c.SDKAppID == 0 || c.SecretKey == "" || c.AdminAccount == "" {
		return "", fmt.Errorf("Tencent SDKAppID, SecretKey and AdminAccount are required")
	}
	ttl := c.SigTTL
	if ttl <= 0 {
		ttl = 86400
	}
	sig, err := tencentyun.GenUserSig(c.SDKAppID, c.SecretKey, c.AdminAccount, ttl)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sdkappid=%d&identifier=%s&usersig=%s&random=%d&contenttype=json", c.SDKAppID, c.AdminAccount, sig, time.Now().UnixNano()%900000000+100000000), nil
}

// UserSig returns a short-lived client credential. SecretKey never leaves the
// Bridge process and is only used for this calculation.
func (c TencentClient) UserSig(userID string) (string, error) {
	if c.SDKAppID == 0 || c.SecretKey == "" || strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("Tencent SDKAppID, SecretKey and user ID are required")
	}
	ttl := c.SigTTL
	if ttl <= 0 {
		ttl = 86400
	}
	return tencentyun.GenUserSig(c.SDKAppID, c.SecretKey, userID, ttl)
}

func (c TencentClient) request(ctx context.Context, service, command string, body any, response any) error {
	query, err := c.authQuery()
	if err != nil {
		return err
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "https://adminapisgp.im.qcloud.com"
	}
	return c.HTTP.DoJSON(ctx, http.MethodPost, base+"/v4/"+service+"/"+command+"?"+query, "", body, response)
}

func (c TencentClient) EnsureUser(ctx context.Context, identity domain.Identity) (domain.Identity, error) {
	// Tencent UserID is limited to 32 characters. TangSeng UIDs are commonly
	// 32 hex characters, so the `dd_` shadow prefix must be shortened before
	// account import.
	if len(identity.TargetID) > 32 {
		identity.TargetID = identity.TargetID[:32]
	}
	var response map[string]any
	// A group/message can contain only an ID. Empty optional fields must not
	// erase a profile previously synchronized from the account's source.
	body := map[string]any{"Identifier": identity.TargetID}
	if identity.DisplayName != "" {
		body["Nick"] = identity.DisplayName
	}
	if identity.AvatarURL != "" {
		body["FaceUrl"] = identity.AvatarURL
	}
	if err := c.request(ctx, "im_open_login_svc", "account_import", body, &response); err != nil {
		return identity, err
	}
	if code, ok := response["ErrorCode"].(float64); ok && int(code) != 0 {
		if info, _ := response["ErrorInfo"].(string); info != "" {
			return identity, fmt.Errorf("Tencent account import failed: %d %s", int(code), info)
		}
		return identity, fmt.Errorf("Tencent account import failed: %d", int(code))
	}
	return identity, nil
}

func (c TencentClient) EnsureGroup(ctx context.Context, identity domain.Identity) (domain.Identity, error) {
	var response map[string]any
	return identity, c.request(ctx, "group_open_http_svc", "create_group", map[string]any{
		"Type": "Private", "Name": identity.DisplayName, "GroupId": identity.TargetID,
	}, &response)
}

// AddFriend links a Huawei Tencent account with a TangSeng shadow account
// after the TangSeng user accepts the corresponding request.
func (c TencentClient) AddFriend(ctx context.Context, fromID, toID string) error {
	return c.FriendAdd(ctx, fromID, toID, "", true)
}

func (c TencentClient) SendText(ctx context.Context, event domain.MessageEvent, from, to domain.Identity, marker string) (string, error) {
	var response struct {
		ActionStatus string `json:"ActionStatus"`
		ErrorCode    int    `json:"ErrorCode"`
		ErrorInfo    string `json:"ErrorInfo"`
		MsgKey       string `json:"MsgKey"`
		MsgSeq       int64  `json:"MsgSeq"`
	}
	metadata := map[string]any{"bridge_source": string(event.Source), "bridge_event_id": marker, "bridge_message_id": event.SourceMessageID}
	if event.Reply != nil {
		metadata["messageReply"] = event.TencentReply()
	}
	cloudData, _ := json.Marshal(metadata)
	content := domain.NormalizeEmojiText(event.Content)
	random := stableMessageRandom(event.SourceMessageID, marker, from.TargetID, to.TargetID)
	body := map[string]any{"From_Account": from.TargetID, "MsgRandom": random, "CloudCustomData": string(cloudData), "MsgBody": []any{map[string]any{"MsgType": "TIMTextElem", "MsgContent": map[string]any{"Text": content}}}}
	if len(event.Parts) > 0 {
		body["MsgBody"] = domain.TencentElements(event.Parts)
	}
	service, command := "openim", "sendmsg"
	if event.ConversationType == domain.ConversationGroup {
		service, command = "group_open_http_svc", "send_group_msg"
		body = map[string]any{"GroupId": to.TargetID, "From_Account": from.TargetID, "Random": random, "CloudCustomData": string(cloudData), "MsgBody": body["MsgBody"]}
		if event.MirrorSender != "" {
			body["From_Account"] = event.MirrorSender
		}
		if event.Mention != nil {
			at := []any{}
			if event.Mention.All == 1 {
				at = append(at, map[string]any{"GroupAtAllFlag": 1})
			}
			for _, uid := range event.Mention.UIDs {
				at = append(at, map[string]any{"GroupAtAllFlag": 0, "GroupAt_Account": uid})
			}
			if len(at) > 0 {
				body["GroupAtInfo"] = at
			}
		}
	} else {
		body["To_Account"] = to.TargetID
		if event.NeedReceipt {
			body["IsNeedReadReceipt"] = 1
		}
	}
	if len(event.Parts) == 1 && event.Parts[0].Kind == "tip" {
		body["SendMsgControl"] = []string{"NoUnread"}
		body["ForbidCallbackControl"] = []string{"ForbidBeforeSendMsgCallback", "ForbidAfterSendMsgCallback"}
	}
	body["OfflinePushInfo"] = c.offlinePush(event, from, to)
	if err := c.request(ctx, service, command, body, &response); err != nil {
		return "", err
	}
	if response.ActionStatus != "OK" || response.ErrorCode != 0 {
		return "", fmt.Errorf("Tencent REST failed: %d %s", response.ErrorCode, response.ErrorInfo)
	}
	if response.MsgKey != "" {
		return response.MsgKey, nil
	}
	if event.ConversationType == domain.ConversationGroup && response.MsgSeq > 0 {
		return fmt.Sprintf("%s:%d", to.TargetID, response.MsgSeq), nil
	}
	return domain.StableID(string(event.Source), event.SourceMessageID, to.TargetID), nil
}

func stableMessageRandom(parts ...string) int64 {
	value := domain.StableID(parts...)
	n, err := strconv.ParseUint(value[:8], 16, 32)
	n &= 0x7fffffff
	if err != nil || n == 0 {
		return 1
	}
	return int64(n)
}

type TangSengClient struct {
	Credential                                                                     func(context.Context, string) (UserCredential, bool, error)
	RefreshUserIfToken                                                             func(context.Context, string, string) error
	RefreshUser                                                                    func(context.Context, string) error
	BaseURL, Token, UserEndpoint, LoginEndpoint, GroupEndpoint, SendEndpoint       string
	ShadowPassword, LegacyShadowPassword, Manufacturer, InviteCode, ShadowOwnerUID string
	Username, Password, DeviceID                                                   string
	HTTP                                                                           HTTPClient
	UserTokens                                                                     *sync.Map
	Sessions                                                                       *proxy.WuKongManager
}

type SearchUser struct {
	UID     string `json:"uid"`
	Name    string `json:"name"`
	Nick    string `json:"nick"`
	Avatar  string `json:"avatar"`
	FaceURL string `json:"face_url"`
	Vercode string `json:"vercode"`
}

// SearchUser performs the public TangSeng user search used by the bridge
// contact flow. The response parser accepts both a top-level list and the
// common {data:{users:[...]}} wrapper used by deployed versions.
func (c *TangSengClient) SearchUser(ctx context.Context, keyword string) ([]SearchUser, error) {
	if c == nil || c.BaseURL == "" {
		return nil, fmt.Errorf("TANGSENG_BASE_URL is not configured")
	}
	url := c.endpoint("/v1/user/search") + "?keyword=" + url.QueryEscape(strings.TrimSpace(keyword))
	var raw json.RawMessage
	if err := c.HTTP.DoJSON(ctx, http.MethodGet, url, "", nil, &raw); err != nil {
		return nil, err
	}
	return parseSearchUsers(raw)
}

func parseSearchUsers(raw json.RawMessage) ([]SearchUser, error) {
	var direct []SearchUser
	if json.Unmarshal(raw, &direct) == nil && direct != nil {
		return direct, nil
	}
	var wrapped struct {
		Data  json.RawMessage `json:"data"`
		Users []SearchUser    `json:"users"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, err
	}
	if len(wrapped.Users) > 0 {
		return wrapped.Users, nil
	}
	if len(wrapped.Data) > 0 {
		if json.Unmarshal(wrapped.Data, &direct) == nil {
			return direct, nil
		}
		// Deployed TangSeng returns a single user object for an exact
		// username/UID match: {"data":{"uid":...,"name":...},"exist":1}.
		var single SearchUser
		if json.Unmarshal(wrapped.Data, &single) == nil && single.UID != "" {
			return []SearchUser{single}, nil
		}
		var nested struct {
			Users []SearchUser `json:"users"`
		}
		if json.Unmarshal(wrapped.Data, &nested) == nil {
			return nested.Users, nil
		}
	}
	return []SearchUser{}, nil
}

// FriendApply executes the request as the Huawei user's own TangSeng shadow
// account. It deliberately never uses the service/admin token.
func (c *TangSengClient) FriendApply(ctx context.Context, ownerShadowUID, targetUID, remark, vercode string) error {
	if c == nil || c.BaseURL == "" {
		return fmt.Errorf("TANGSENG_BASE_URL is not configured")
	}
	body := map[string]any{"to_uid": targetUID, "remark": remark, "vercode": vercode}
	return c.UserRequest(ctx, ownerShadowUID, http.MethodPost, "/v1/friend/apply", body, nil)
}

// Login connects a configured real test/service account to the public TangSeng
// login API. It deliberately uses the same PC(2) credentials as the WuKong
// websocket session and does not require any TangSeng server changes.
func (c *TangSengClient) Login(ctx context.Context) (string, error) {
	if c == nil || c.BaseURL == "" || c.Username == "" || c.Password == "" {
		return "", fmt.Errorf("TangSeng test account credentials are not configured")
	}
	deviceID := c.DeviceID
	if deviceID == "" {
		deviceID = "hwbbridge-" + c.Username
	}
	body := map[string]any{
		"username": c.Username,
		"password": c.Password,
		"flag":     2,
		"device": map[string]any{
			"device_id": deviceID, "device_name": "Huawei Bridge", "device_model": "bridge",
		},
	}
	var response struct {
		UID   string `json:"uid"`
		Token string `json:"token"`
		Data  struct {
			UID   string `json:"uid"`
			Token string `json:"token"`
		} `json:"data"`
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	if err := c.HTTP.DoJSON(ctx, http.MethodPost, c.endpoint(c.LoginEndpoint), "", body, &response); err != nil {
		return "", err
	}
	if response.UID == "" {
		response.UID = response.Data.UID
	}
	if response.Token == "" {
		response.Token = response.Data.Token
	}
	if response.UID == "" || response.Token == "" {
		return "", fmt.Errorf("TangSeng login returned no uid/token: status=%d message=%s", response.Status, response.Message)
	}
	if c.UserTokens == nil {
		c.UserTokens = &sync.Map{}
	}
	c.UserTokens.Store(response.UID, response.Token)
	if c.Sessions != nil {
		c.Sessions.SetToken(response.UID, response.Token)
		c.Sessions.SetDeviceID(response.UID, deviceID)
	}
	return response.UID, nil
}

func (c TangSengClient) endpoint(path string) string {
	return strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func (c TangSengClient) EnsureUser(ctx context.Context, identity domain.Identity) (domain.Identity, error) {
	if c.Credential != nil {
		cred, found, err := c.Credential(ctx, identity.SourceID)
		if err != nil {
			return identity, err
		}
		if found {
			return c.restoreNamedUser(ctx, identity, cred)
		}
	}
	if c.BaseURL == "" {
		return identity, fmt.Errorf("TANGSENG_BASE_URL is not configured")
	}
	// TargetID is replaced with TangSeng's 32 byte UID after the first
	// provision.  It therefore cannot be reused as the username during a
	// restart.  Always derive the stable shadow username from the immutable
	// Tencent source ID.
	username := domain.ShadowID(identity.Source, identity.SourceID)
	if len(username) < 8 {
		username = "bridge_" + username
	}
	if len(username) > 22 {
		username = "tx_" + domain.StableID(string(identity.Source), identity.SourceID)[:19]
	}
	body := map[string]any{"username": username, "name": identity.DisplayName, "password": c.ShadowPassword, "flag": 2, "device": map[string]any{"device_id": "hwbridge-" + username, "device_name": "Huawei Bridge", "device_model": "bridge"}, "manufacturer": c.Manufacturer, "invite_code": c.InviteCode}
	var response struct {
		UID   string `json:"uid"`
		Token string `json:"token"`
		Data  struct {
			UID   string `json:"uid"`
			Token string `json:"token"`
		} `json:"data"`
	}
	err := c.HTTP.DoJSON(ctx, http.MethodPost, c.endpoint(c.UserEndpoint), "", body, &response)
	// Registration is idempotent from the bridge's perspective. An already
	// existing shadow username is logged in below to recover its current UID.
	if err != nil && !isAlreadyRegistered(err) {
		return identity, nativeFailure(err)
	}
	if response.UID == "" {
		response.UID = response.Data.UID
	}
	if response.Token == "" {
		response.Token = response.Data.Token
	}
	if response.UID != "" {
		identity.TargetID = response.UID
	}
	if response.Token == "" {
		var login struct {
			UID   string `json:"uid"`
			Token string `json:"token"`
			Data  struct {
				UID   string `json:"uid"`
				Token string `json:"token"`
			} `json:"data"`
		}
		loginBody := map[string]any{"username": username, "password": c.ShadowPassword, "flag": 2, "device": map[string]any{"device_id": "hwbridge-" + username, "device_name": "Huawei Bridge", "device_model": "bridge"}}
		loginErr := c.HTTP.DoJSON(ctx, http.MethodPost, c.endpoint(c.LoginEndpoint), "", loginBody, &login)
		// Older shadow accounts were created before the minimum password rule
		// was enforced. Keep the current password for new accounts, but allow
		// one configured legacy password when restoring those accounts.
		if loginErr != nil && c.LegacyShadowPassword != "" && c.LegacyShadowPassword != c.ShadowPassword {
			login = struct {
				UID   string `json:"uid"`
				Token string `json:"token"`
				Data  struct {
					UID   string `json:"uid"`
					Token string `json:"token"`
				} `json:"data"`
			}{}
			loginBody["password"] = c.LegacyShadowPassword
			loginErr = c.HTTP.DoJSON(ctx, http.MethodPost, c.endpoint(c.LoginEndpoint), "", loginBody, &login)
		}
		if loginErr != nil {
			return identity, nativeFailure(loginErr)
		}
		if login.UID == "" {
			login.UID = login.Data.UID
		}
		if login.Token == "" {
			login.Token = login.Data.Token
		}
		if login.UID != "" {
			identity.TargetID = login.UID
		}
		response.Token = login.Token
	}
	if response.Token != "" && c.UserTokens != nil {
		c.UserTokens.Store(identity.TargetID, response.Token)
	}
	if response.Token != "" && c.Sessions != nil {
		c.Sessions.SetToken(identity.TargetID, response.Token)
	}
	return identity, nil
}

func isAlreadyRegistered(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "exist") || strings.Contains(message, "already") || strings.Contains(message, "已存在") || strings.Contains(message, "用户名已存在")
}

func (c TangSengClient) EnsureGroup(ctx context.Context, identity domain.Identity) (domain.Identity, error) {
	if c.BaseURL == "" {
		return identity, fmt.Errorf("TANGSENG_BASE_URL is not configured")
	}
	members := []string{}
	if c.ShadowOwnerUID != "" {
		members = append(members, c.ShadowOwnerUID)
	}
	body := map[string]any{"name": identity.DisplayName, "members": members}
	var response struct {
		GroupNo string `json:"group_no"`
		Data    struct {
			GroupNo string `json:"group_no"`
		} `json:"data"`
	}
	if err := c.HTTP.DoJSON(ctx, http.MethodPost, c.endpoint(c.GroupEndpoint), c.Token, body, &response); err != nil {
		return identity, err
	}
	if response.GroupNo != "" {
		identity.TargetID = response.GroupNo
	} else if response.Data.GroupNo != "" {
		identity.TargetID = response.Data.GroupNo
	}
	return identity, nil
}

func (c TangSengClient) SendText(ctx context.Context, event domain.MessageEvent, from, to domain.Identity, marker string) (string, error) {
	if c.BaseURL == "" {
		return "", fmt.Errorf("TANGSENG_BASE_URL is not configured")
	}
	senderToken := c.Token
	if c.UserTokens != nil {
		senderToken = ""
		if value, ok := c.UserTokens.Load(from.TargetID); ok {
			senderToken, _ = value.(string)
		}
	}
	if senderToken == "" {
		return "", fmt.Errorf("TangSeng sender token is not configured for %s", from.TargetID)
	}
	if c.Sessions != nil {
		channelType := 1
		if event.ConversationType == domain.ConversationGroup {
			channelType = 2
		}
		payload, payloadErr := json.Marshal(event.TangPayload())
		if payloadErr != nil {
			return "", payloadErr
		}
		id, err := c.Sessions.SendPayloadWithReceipt(ctx, from.TargetID, to.TargetID, channelType, payload, "hwb_"+domain.StableID(marker, event.SourceMessageID), event.NeedReceipt)
		var failure *domain.DeliveryError
		if errors.As(err, &failure) && failure.Auth && c.RefreshUserIfToken != nil {
			if refreshErr := c.RefreshUserIfToken(ctx, from.TargetID, senderToken); refreshErr != nil {
				return "", refreshErr
			}
			return c.Sessions.SendPayloadWithReceipt(ctx, from.TargetID, to.TargetID, channelType, payload, "hwb_"+domain.StableID(marker, event.SourceMessageID), event.NeedReceipt)
		}
		return id, err
	}
	payload := event.TangPayload()
	payload["bridge_source"] = string(event.Source)
	payload["bridge_event_id"] = marker
	body := map[string]any{"token": senderToken, "receive_channel_id": to.TargetID, "receive_channel_type": 1, "payload": payload}
	if event.ConversationType == domain.ConversationGroup {
		body["receive_channel_type"] = 2
	}
	var response struct {
		MessageID string `json:"message_id"`
		MsgID     string `json:"msg_id"`
		ID        string `json:"id"`
	}
	if err := c.HTTP.DoJSON(ctx, http.MethodPost, c.endpoint(c.SendEndpoint), senderToken, body, &response); err != nil {
		return "", err
	}
	for _, id := range []string{response.MessageID, response.MsgID, response.ID} {
		if id != "" {
			return id, nil
		}
	}
	return domain.StableID(string(event.Source), event.SourceMessageID, to.TargetID), nil
}
