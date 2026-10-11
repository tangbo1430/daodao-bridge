package account

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/security"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type PrepareRequest struct {
	RequestID string `json:"request_id"`
	Username  string `json:"username"`
	Nick      string `json:"nick"`
	Invite    string `json:"invite_code"`
	ClientIP  string `json:"client_ip"`
}
type Reservation struct {
	ID          string `json:"reserve_id"`
	RequestID   string `json:"request_id"`
	Username    string `json:"username"`
	Nick        string `json:"nick"`
	UID         string `json:"tsdd_uid"`
	AppID       string `json:"app_id"`
	TIMUID      string `json:"tim_uid,omitempty"`
	Status      string `json:"status"`
	Cipher      string `json:"password_cipher"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}
type RegistrationError struct {
	Code    int
	Message string
}

func (e *RegistrationError) Error() string         { return e.Message }
func registrationError(code int, msg string) error { return &RegistrationError{code, msg} }

var newUsername = regexp.MustCompile(`^(?:[A-Za-z][A-Za-z0-9_]{7,21}|[0-9]{8,22})$`)

func (s *Service) ConfigureRegistration(secret string) {
	if secret == "" {
		return
	}
	key := sha256.Sum256([]byte("hw-bridge-registration-v1\x00" + secret))
	s.RegistrationBox, _ = security.NewSecretBox(key[:])
}
func (s *Service) reservation(ctx context.Context, id string) (Reservation, error) {
	c, err := s.Store.GetCursor(ctx, "registration:"+id)
	if err != nil {
		return Reservation{}, err
	}
	var r Reservation
	err = json.Unmarshal([]byte(c.Aux), &r)
	return r, err
}
func (s *Service) saveReservation(ctx context.Context, r *Reservation) error {
	r.UpdatedAt = time.Now().Unix()
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return s.Store.PutCursor(ctx, domain.SyncCursor{Key: "registration:" + r.ID, Aux: string(b), Value: r.UpdatedAt, StartedAt: r.CreatedAt})
}
func (s *Service) credential(r Reservation) (adapter.UserCredential, error) {
	if s.RegistrationBox == nil {
		return adapter.UserCredential{}, registrationError(1001, "注册加密配置未就绪")
	}
	p, e := s.RegistrationBox.Decrypt(r.Cipher)
	if e != nil {
		return adapter.UserCredential{}, registrationError(1005, "注册凭据读取失败")
	}
	return adapter.UserCredential{Username: r.Username, Password: p, UID: r.UID}, nil
}

func (s *Service) Prepare(ctx context.Context, q PrepareRequest) (Reservation, error) {
	s.registrationMu.Lock()
	defer s.registrationMu.Unlock()
	if s.Store == nil || s.Tang == nil || s.RegistrationBox == nil {
		return Reservation{}, registrationError(1001, "注册服务未配置")
	}
	if len(q.RequestID) < 8 || len(q.RequestID) > 128 {
		return Reservation{}, registrationError(1002, "注册请求标识无效")
	}
	if !newUsername.MatchString(q.Username) {
		return Reservation{}, registrationError(1002, "用户名须为8至22位；可使用纯数字，或以英文字母开头的字母、数字和下划线")
	}
	if net.ParseIP(q.ClientIP) == nil {
		return Reservation{}, registrationError(1002, "注册来源地址无效")
	}
	fingerprint := domain.StableID(q.Username, q.Nick, q.Invite)
	id := domain.StableID("registration", q.RequestID)
	r, err := s.reservation(ctx, id)
	recoverLogin := false
	if err == nil {
		if r.Fingerprint != fingerprint {
			return Reservation{}, registrationError(1002, "同一请求不能修改注册信息")
		}
		if r.Status == "prepared" || r.Status == "committed" {
			return r, nil
		}
		if r.Status == "aborted" || r.Status == "abandoned" || r.Status == "rejected" {
			return Reservation{}, registrationError(1003, "该注册请求已结束，请重新发起")
		}
		recoverLogin = true
	} else {
		if !errors.Is(err, store.ErrNotFound) {
			return Reservation{}, err
		}
		// Channel enrollment is an operational credential, not the optional
		// product referral entered by an APK user. Existing requests can recover
		// even when the channel credential has subsequently been rotated.
		if strings.TrimSpace(q.Invite) == "" && strings.TrimSpace(s.Tang.InviteCode) == "" {
			return Reservation{}, registrationError(1001, "注册服务暂不可用，请稍后重试")
		}
		if err = s.registrationRate(ctx, q.ClientIP); err != nil {
			return Reservation{}, err
		}
		r = Reservation{ID: id, RequestID: q.RequestID, Username: q.Username, Nick: q.Nick, Fingerprint: fingerprint, Status: "registering", CreatedAt: time.Now().Unix()}
		index, e := s.Store.GetCursor(ctx, "registration-name:"+strings.ToLower(q.Username))
		if e == nil {
			old, e := s.reservation(ctx, index.Aux)
			if e != nil {
				return r, e
			}
			if old.Status != "aborted" && old.Status != "abandoned" && old.Status != "rejected" {
				return r, registrationError(1003, "用户名已被占用或正在注册")
			}
			if old.Status != "rejected" {
				r.Cipher, r.UID, r.AppID = old.Cipher, old.UID, old.AppID
				recoverLogin = true
			}
		} else if !errors.Is(e, store.ErrNotFound) {
			return r, e
		}
		if r.Cipher == "" {
			b := make([]byte, 12)
			if _, e = rand.Read(b); e != nil {
				return r, e
			}
			r.Cipher, e = s.RegistrationBox.Encrypt(hex.EncodeToString(b))
			if e != nil {
				return r, e
			}
		}
		if e = s.saveReservation(ctx, &r); e != nil {
			return r, e
		}
		if e = s.Store.PutCursor(ctx, domain.SyncCursor{Key: "registration-name:" + strings.ToLower(q.Username), Aux: r.ID}); e != nil {
			return r, e
		}
	}
	cred, err := s.credential(r)
	if err != nil {
		return r, err
	}
	if !recoverLogin {
		invite := strings.TrimSpace(q.Invite)
		if invite == "" {
			invite = s.Tang.InviteCode
		}
		native, e := s.Tang.RegisterNamedUser(ctx, cred, q.Nick, invite)
		if e != nil {
			var status *adapter.HTTPStatusError
			if errors.As(e, &status) && status.Status >= 400 && status.Status < 500 && status.Status != 408 && status.Status != 429 {
				r.Status = "rejected"
				if save := s.saveReservation(ctx, &r); save != nil {
					return r, save
				}
				if strings.TrimSpace(q.Invite) == "" && strings.Contains(status.Message, "邀请码") {
					return r, registrationError(1001, "注册服务暂不可用，请稍后重试")
				}
				return r, registrationNativeError(status)
			}
			return r, registrationError(1005, "注册结果待确认，请使用同一请求重试")
		}
		r.UID, r.AppID = native.UID, native.AppID
		cred.UID = r.UID
		if err = s.saveReservation(ctx, &r); err != nil {
			return r, err
		}
	}
	native, err := s.Tang.LoginNamedUser(ctx, cred)
	if err != nil {
		return r, registrationError(1005, "注册身份确认失败，请稍后使用同一请求重试")
	}
	if r.UID == "" {
		r.UID = native.UID
		cred.UID = r.UID
		native, err = s.Tang.LoginNamedUser(ctx, cred)
		if err != nil {
			return r, registrationError(1005, "注册登录待确认")
		}
	}
	if native.UID != r.UID {
		return r, registrationError(1005, "注册账号身份不匹配")
	}
	r.Status = "prepared"
	err = s.saveReservation(ctx, &r)
	return r, err
}

func registrationNativeError(s *adapter.HTTPStatusError) error {
	switch {
	case strings.Contains(s.Message, "已存在"):
		return registrationError(1003, "用户名已被占用，请换一个")
	case strings.Contains(s.Message, "邀请码"):
		return registrationError(1004, "邀请码无效或已失效")
	case strings.Contains(s.Message, "用户名"):
		return registrationError(1002, "用户名格式不符合原生注册规则")
	}
	return registrationError(1001, "注册通道暂不可用")
}

func (s *Service) registrationRate(ctx context.Context, ip string) error {
	now := time.Now().UTC()
	for _, limit := range []struct {
		key string
		max int64
	}{{"registration-ip:" + domain.StableID(ip) + ":" + now.Format("2006010215"), 5}, {"registration-global:" + now.Format("200601021504"), 60}} {
		c, e := s.Store.GetCursor(ctx, limit.key)
		if e != nil && !errors.Is(e, store.ErrNotFound) {
			return e
		}
		if c.Value >= limit.max {
			return registrationError(1005, "注册请求过于频繁，请稍后重试")
		}
		c.Key = limit.key
		c.Value++
		if e = s.Store.PutCursor(ctx, c); e != nil {
			return e
		}
	}
	return nil
}

func (s *Service) CommitRegistration(ctx context.Context, id, tim string) error {
	s.registrationMu.Lock()
	defer s.registrationMu.Unlock()
	r, err := s.reservation(ctx, id)
	if err != nil {
		return err
	}
	if tim == "" || len(tim) > 32 {
		return registrationError(1002, "本地账号标识无效")
	}
	if r.TIMUID != "" && r.TIMUID != tim {
		return registrationError(1003, "注册预留已属于其他账号")
	}
	if r.Status == "committed" {
		return nil
	}
	if r.Status != "prepared" && r.Status != "committing" {
		return registrationError(1005, "注册预留尚未就绪或已取消")
	}
	if old, e := s.Store.FindIdentity(ctx, domain.SourceTencent, tim); e == nil && (old.TargetID != r.UID || old.Target != domain.SourceTang) {
		return registrationError(1003, "已有账号映射不能覆盖")
	} else if e != nil && !errors.Is(e, store.ErrNotFound) {
		return e
	}
	state, e := s.Store.GetCursor(ctx, "account:"+tim)
	if e != nil && !errors.Is(e, store.ErrNotFound) {
		return e
	}
	if state.Aux == "deleted" || state.Aux == "deleting" || state.Aux == "banned" || state.Aux == "tenant_disabled" {
		return registrationError(1003, "当前账号状态不允许重新确认注册")
	}
	r.TIMUID = tim
	r.Status = "committing"
	if err = s.saveReservation(ctx, &r); err != nil {
		return err
	}
	cred, err := s.credential(r)
	if err != nil {
		return err
	}
	native, err := s.Tang.LoginNamedUser(ctx, cred)
	if err != nil {
		return registrationError(1005, "账号开通中，请稍后重试")
	}
	s.Tang.UserTokens.Store(r.UID, native.Token)
	if err = s.Tang.UserRequest(ctx, r.UID, http.MethodPut, "/v1/user/current", map[string]string{"name": r.Nick}, nil); err != nil {
		s.Tang.UserTokens.Delete(r.UID)
		return registrationError(1005, "注册资料确认待重试")
	}
	if err = s.Store.PutCursor(ctx, domain.SyncCursor{Key: "account-registration:" + tim, Aux: r.ID}); err != nil {
		return err
	}
	if err = s.persistStatus(ctx, tim, "registration_pending"); err != nil {
		return err
	}
	identity := domain.Identity{Source: domain.SourceTencent, SourceID: tim, Target: domain.SourceTang, TargetID: r.UID, DisplayName: r.Nick, EntityType: "user", Shadow: true}
	if err = s.Store.UpsertIdentity(ctx, identity); err != nil {
		return err
	}
	s.Tang.UserTokens.Store(r.UID, native.Token)
	if s.Tang.Sessions != nil {
		s.Tang.Sessions.SetDeviceID(r.UID, "hwbridge-"+r.UID)
		s.Tang.Sessions.SetToken(r.UID, native.Token)
	}
	if err = s.persistLogin(ctx, tim); err != nil {
		return err
	}
	s.mu.Lock()
	s.records[tim] = Record{Identity: identity, Status: "reserved", ReservedAt: r.CreatedAt, LastLoginAt: time.Now().Unix()}
	s.mu.Unlock()
	if _, err = s.Confirm(ctx, tim); err != nil {
		return err
	}
	r.Status = "committed"
	return s.saveReservation(ctx, &r)
}

func (s *Service) AbortRegistration(ctx context.Context, id string) error {
	s.registrationMu.Lock()
	defer s.registrationMu.Unlock()
	r, err := s.reservation(ctx, id)
	if err != nil {
		return err
	}
	if r.Status == "committed" || r.Status == "committing" {
		return registrationError(1003, "已确认的注册不能取消")
	}
	if r.Status == "aborted" || r.Status == "abandoned" {
		return nil
	}
	r.Status = "aborted"
	return s.saveReservation(ctx, &r)
}

func (s *Service) RegisteredCredential(ctx context.Context, tim string) (adapter.UserCredential, bool, error) {
	c, err := s.Store.GetCursor(ctx, "account-registration:"+tim)
	if errors.Is(err, store.ErrNotFound) {
		return adapter.UserCredential{}, false, nil
	}
	if err != nil {
		return adapter.UserCredential{}, true, err
	}
	r, err := s.reservation(ctx, c.Aux)
	if err != nil {
		return adapter.UserCredential{}, true, err
	}
	if r.TIMUID != tim || (r.Status != "committed" && r.Status != "committing") {
		return adapter.UserCredential{}, true, fmt.Errorf("registration credential inactive")
	}
	cred, err := s.credential(r)
	return cred, true, err
}

func (s *Service) ReconcileRegistrations(ctx context.Context) error {
	if s.Store == nil || s.RegistrationBox == nil {
		return nil
	}
	last, e := s.Store.GetCursor(ctx, "registration-scan")
	if e != nil && !errors.Is(e, store.ErrNotFound) {
		return e
	}
	if time.Now().Unix()-last.Value < 600 {
		return nil
	}
	if e = s.Store.PutCursor(ctx, domain.SyncCursor{Key: "registration-scan", Value: time.Now().Unix()}); e != nil {
		return e
	}
	rows, err := s.Store.ListCursors(ctx, "registration:")
	if err != nil {
		return err
	}
	for _, row := range rows {
		var r Reservation
		if err = json.Unmarshal([]byte(row.Aux), &r); err != nil {
			return err
		}
		if time.Now().Unix()-r.UpdatedAt < 600 {
			continue
		}
		if r.Status == "registering" {
			// A lost register response is recovered ONLY by the saved random
			// credential; never register again or adopt an existing username.
			if err = s.recoverReservation(ctx, r.ID); err != nil {
				continue
			}
			r, err = s.reservation(ctx, r.ID)
			if err != nil {
				return err
			}
		}
		if (r.Status == "prepared" || r.Status == "committing") && s.RegistrationLookup != nil {
			tim, e := s.RegistrationLookup(ctx, r.Username, r.ID)
			if e != nil {
				return e
			}
			if tim != "" {
				if e = s.CommitRegistration(ctx, r.ID, tim); e != nil {
					return e
				}
				continue
			} else if r.Status == "prepared" && time.Now().Unix()-r.CreatedAt > 86400 {
				if e = s.AbortRegistration(ctx, r.ID); e != nil {
					return e
				}
				r, e = s.reservation(ctx, r.ID)
				if e != nil {
					return e
				}
			}
		}
		if r.Status == "aborted" && time.Now().Unix()-r.CreatedAt > 86400 && r.UID != "" {
			s.registrationMu.Lock()
			r, err = s.reservation(ctx, r.ID)
			if err != nil {
				s.registrationMu.Unlock()
				return err
			}
			index, e := s.Store.GetCursor(ctx, "registration-name:"+strings.ToLower(r.Username))
			if e == nil && index.Aux == r.ID && r.Status == "aborted" {
				cred, e := s.credential(r)
				if e == nil {
					var n adapter.NativeAccount
					n, e = s.Tang.LoginNamedUser(ctx, cred)
					if e == nil {
						s.Tang.UserTokens.Store(r.UID, n.Token)
						e = s.Tang.UserRequest(ctx, r.UID, http.MethodPut, "/v1/user/current", map[string]string{"name": "未完成注册"}, nil)
						s.Tang.UserTokens.Delete(r.UID)
					}
				}
				if e == nil {
					r.Status = "abandoned"
					e = s.saveReservation(ctx, &r)
				}
				err = e
			}
			s.registrationMu.Unlock()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) recoverReservation(ctx context.Context, id string) error {
	s.registrationMu.Lock()
	defer s.registrationMu.Unlock()
	r, err := s.reservation(ctx, id)
	if err != nil || r.Status != "registering" {
		return err
	}
	cred, err := s.credential(r)
	if err != nil {
		return err
	}
	n, err := s.Tang.LoginNamedUser(ctx, cred)
	if err != nil {
		return err
	}
	r.UID, r.AppID, r.Status = n.UID, n.AppID, "prepared"
	return s.saveReservation(ctx, &r)
}

// Erase all generations of this credential, including a reused aborted reservation.
// Keep the account-registration index so no legacy fallback can recreate the user.
func (s *Service) EraseRegistrationCredential(ctx context.Context, tim string) error {
	if s.Store == nil {
		return nil
	}
	s.registrationMu.Lock()
	defer s.registrationMu.Unlock()
	index, err := s.Store.GetCursor(ctx, "account-registration:"+tim)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	r, err := s.reservation(ctx, index.Aux)
	if err != nil {
		return err
	}
	rows, err := s.Store.ListCursors(ctx, "registration:")
	if err != nil {
		return err
	}
	for _, row := range rows {
		var old Reservation
		if err = json.Unmarshal([]byte(row.Aux), &old); err != nil {
			return err
		}
		if old.UID != r.UID {
			continue
		}
		old.Cipher, old.Status = "", "deleted"
		if err = s.saveReservation(ctx, &old); err != nil {
			return err
		}
	}
	return nil
}
