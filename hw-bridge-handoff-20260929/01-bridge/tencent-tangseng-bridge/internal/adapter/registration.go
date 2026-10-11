package adapter

import (
	"context"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
)

type UserCredential struct{ Username, Password, UID string }
type NativeAccount struct {
	UID   string `json:"uid"`
	AppID string `json:"app_id"`
	Token string `json:"token"`
}

func (c *TangSengClient) nativeAccount(ctx context.Context, path string, body any) (NativeAccount, error) {
	var r struct {
		NativeAccount
		Data    NativeAccount `json:"data"`
		Status  int           `json:"status"`
		Message string        `json:"msg"`
	}
	if err := c.HTTP.DoJSON(ctx, http.MethodPost, c.endpoint(path), "", body, &r); err != nil {
		return NativeAccount{}, err
	}
	if r.Status >= 400 {
		return NativeAccount{}, &HTTPStatusError{r.Status, r.Message}
	}
	if r.UID == "" {
		r.NativeAccount = r.Data
	}
	if r.UID == "" {
		return NativeAccount{}, fmt.Errorf("native account response missing identity")
	}
	return r.NativeAccount, nil
}

func (c *TangSengClient) RegisterNamedUser(ctx context.Context, cred UserCredential, nick, invite string) (NativeAccount, error) {
	return c.nativeAccount(ctx, c.UserEndpoint, map[string]any{"username": cred.Username, "password": cred.Password, "name": nick, "invite_code": invite, "flag": 2, "manufacturer": "huawei"})
}

func (c *TangSengClient) LoginNamedUser(ctx context.Context, cred UserCredential) (NativeAccount, error) {
	device := cred.UID
	if device == "" {
		device = cred.Username
	}
	device = "hwbridge-" + device
	// Captured on the deployed native API: the plan's 41-character ID fails
	// with "更新用户登录设备失败"; the same login with 40 characters succeeds.
	// This changes HTTP device metadata only, not the native UID or PC(2) UID.
	if len(device) > 40 {
		device = "hwbridge" + cred.UID
		if len(device) > 40 {
			device = "hwbridge" + domain.StableID(cred.UID)[:32]
		}
	}
	r, err := c.nativeAccount(ctx, c.LoginEndpoint, map[string]any{"username": cred.Username, "password": cred.Password, "flag": 2, "device": map[string]string{"device_id": device, "device_name": "华为互通", "device_model": "hw-bridge"}})
	if err != nil {
		return r, err
	}
	if cred.UID != "" && cred.UID != r.UID {
		return NativeAccount{}, fmt.Errorf("native login identity mismatch")
	}
	if r.Token == "" {
		return NativeAccount{}, fmt.Errorf("native login token missing")
	}
	return r, nil
}

// Reauthenticate never registers. It is used for a mapped user's renewal and
// explicit unban verification, including the unchanged legacy username/password.
func (c *TangSengClient) Reauthenticate(ctx context.Context, id domain.Identity) (domain.Identity, error) {
	if c.Credential != nil {
		cred, found, err := c.Credential(ctx, id.SourceID)
		if err != nil {
			return id, err
		}
		if found {
			return c.restoreNamedUser(ctx, id, cred)
		}
	}
	username := domain.ShadowID(id.Source, id.SourceID)
	if len(username) < 8 {
		username = "bridge_" + username
	}
	if len(username) > 22 {
		username = "tx_" + domain.StableID(string(id.Source), id.SourceID)[:19]
	}
	cred := UserCredential{Username: username, Password: c.ShadowPassword, UID: id.TargetID}
	result, err := c.restoreNamedUser(ctx, id, cred)
	var denial *domain.DeliveryError
	if err != nil && c.LegacyShadowPassword != "" && c.LegacyShadowPassword != c.ShadowPassword && !(errors.As(err, &denial) && (denial.Reason == "account_banned" || denial.Reason == "tenant_disabled")) {
		cred.Password = c.LegacyShadowPassword
		return c.restoreNamedUser(ctx, id, cred)
	}
	return result, err
}

func (c *TangSengClient) restoreNamedUser(ctx context.Context, id domain.Identity, cred UserCredential) (domain.Identity, error) {
	if cred.UID != id.TargetID {
		return id, fmt.Errorf("stored native credential identity mismatch")
	}
	r, err := c.LoginNamedUser(ctx, cred)
	if err != nil {
		return id, nativeFailure(err)
	}
	if c.UserTokens == nil {
		return id, fmt.Errorf("credential runtime unavailable")
	}
	c.UserTokens.Store(r.UID, r.Token)
	if c.Sessions != nil {
		c.Sessions.SetDeviceID(r.UID, "hwbridge-"+r.UID)
		c.Sessions.SetToken(r.UID, r.Token)
	}
	return id, nil
}
