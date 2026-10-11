package account

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/security"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
)

type Request struct {
	TencentUserID string `json:"tencent_user_id"`
	DisplayName   string `json:"display_name"`
	AvatarURL     string `json:"avatar_url"`
}
type Record struct {
	LastLoginAt int64           `json:"last_login_at,omitempty"`
	Identity    domain.Identity `json:"identity"`
	Status      string          `json:"status"`
	ReservedAt  int64           `json:"reserved_at"`
	ConfirmedAt int64           `json:"confirmed_at,omitempty"`
}

// Service implements the reserve/confirm/cancel transaction used by the PHP
// account flow. Durable identity projection is still written by the normal
// Engine inbox path; this short lived reservation prevents duplicate register
// calls while the PHP transaction is completing.
type Service struct {
	RegistrationBox    *security.SecretBox
	RegistrationLookup func(context.Context, string, string) (string, error)
	NotifyStatus       func(context.Context, string, string, int64) error
	VerifyConnection   func(context.Context, string) error
	registrationMu     sync.Mutex
	Tang               *adapter.TangSengClient
	Store              store.Store
	StartProxy         func(context.Context, domain.Identity) error
	StopProxy          func(context.Context, domain.Identity) error
	mu                 sync.Mutex
	refreshMu          sync.Mutex
	refreshCompleted   map[string]time.Time
	refreshResults     map[string]error
	records            map[string]Record
}

func New(tang *adapter.TangSengClient) *Service {
	return &Service{Tang: tang, records: make(map[string]Record)}
}

func (s *Service) SetStore(value store.Store) { s.Store = value }

// Refresh reloads only the actor's own credentials; disabled/deleted identities
// cannot acquire a new token through a delayed business request.
func (s *Service) Refresh(ctx context.Context, tangUID string) error {
	var token string
	if s.Tang != nil && s.Tang.UserTokens != nil {
		v, _ := s.Tang.UserTokens.Load(tangUID)
		token, _ = v.(string)
	}
	return s.RefreshIfToken(ctx, tangUID, token)
}

func (s *Service) RefreshIfToken(ctx context.Context, tangUID, failedToken string) error {
	started := time.Now()
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	if s.refreshCompleted[tangUID].After(started) {
		return s.refreshResults[tangUID]
	}
	if s.Store == nil || s.Tang == nil {
		return fmt.Errorf("account refresh unavailable")
	}
	id, err := s.Store.FindIdentityByTarget(ctx, domain.SourceTang, tangUID)
	if err != nil {
		return err
	}
	r, err := s.Profile(ctx, id.SourceID)
	if err != nil {
		return err
	}
	if r.Status != "confirmed" && r.Status != "deleting" {
		return fmt.Errorf("account is inactive")
	}
	if v, ok := s.Tang.UserTokens.Load(tangUID); ok && v != failedToken {
		return nil
	}
	identity, err := s.Tang.Reauthenticate(ctx, r.Identity)
	if s.refreshCompleted == nil {
		s.refreshCompleted = map[string]time.Time{}
		s.refreshResults = map[string]error{}
	}
	s.refreshCompleted[tangUID], s.refreshResults[tangUID] = time.Now(), err
	if err != nil {
		var denied *domain.DeliveryError
		if errors.As(err, &denied) && (denied.Reason == "account_banned" || denied.Reason == "tenant_disabled") {
			state := "banned"
			if denied.Reason == "tenant_disabled" {
				state = "tenant_disabled"
			}
			if save := s.Deactivate(ctx, id.SourceID, state); save != nil {
				return save
			}
		}
		return err
	}
	if identity.TargetID != tangUID {
		return fmt.Errorf("refreshed identity mismatch")
	}
	if err = s.persistLogin(ctx, id.SourceID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.records[id.SourceID]
	if current.Status != "confirmed" && current.Status != "deleting" {
		s.Tang.UserTokens.Delete(tangUID)
		return fmt.Errorf("account became inactive")
	}
	current.LastLoginAt = time.Now().Unix()
	s.records[id.SourceID] = current
	return nil
}
func (s *Service) UpdateProfile(ctx context.Context, req Request) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[req.TencentUserID]
	if !ok {
		return fmt.Errorf("account not found")
	}
	record.Identity.DisplayName = req.DisplayName
	record.Identity.AvatarURL = req.AvatarURL
	if s.Store != nil {
		if err := s.Store.UpsertIdentity(ctx, record.Identity); err != nil {
			return err
		}
	}
	s.records[req.TencentUserID] = record
	return nil
}
func (s *Service) SetProxyLifecycle(start func(context.Context, domain.Identity) error, stop func(context.Context, domain.Identity) error) {
	s.StartProxy, s.StopProxy = start, stop
}

func (s *Service) Reserve(ctx context.Context, req Request) (Record, error) {
	if req.TencentUserID == "" {
		return Record{}, fmt.Errorf("tencent_user_id is required")
	}
	if s.Store != nil {
		state, err := s.Store.GetCursor(ctx, "account:"+req.TencentUserID)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return Record{}, err
		}
		if state.Aux == "deleted" || state.Aux == "deleting" {
			return Record{}, fmt.Errorf("account has been deleted")
		}
		if state.Aux == "banned" || state.Aux == "tenant_disabled" {
			return Record{}, fmt.Errorf("native account requires status verification")
		}
	}
	s.mu.Lock()
	if value, ok := s.records[req.TencentUserID]; ok && value.Status != "cancelled" {
		s.mu.Unlock()
		return value, nil
	}
	s.mu.Unlock()
	identity := domain.Identity{Source: domain.SourceTencent, SourceID: req.TencentUserID, Target: domain.SourceTang, TargetID: domain.ShadowID(domain.SourceTencent, req.TencentUserID), EntityType: "user", DisplayName: req.DisplayName, AvatarURL: req.AvatarURL, Shadow: true}
	if s.Tang != nil {
		ensured, err := s.Tang.EnsureUser(ctx, identity)
		if err != nil {
			return Record{}, err
		}
		identity = ensured
	}
	record := Record{Identity: identity, Status: "reserved", ReservedAt: time.Now().Unix(), LastLoginAt: time.Now().Unix()}
	if err := s.persistLogin(ctx, req.TencentUserID); err != nil {
		return Record{}, err
	}
	if s.Store != nil {
		if err := s.Store.UpsertIdentity(ctx, identity); err != nil {
			return Record{}, err
		}
	}
	s.mu.Lock()
	s.records[req.TencentUserID] = record
	s.mu.Unlock()
	return record, nil
}
func (s *Service) Confirm(ctx context.Context, id string) (Record, error) {
	s.mu.Lock()
	value, ok := s.records[id]
	if !ok {
		s.mu.Unlock()
		return Record{}, fmt.Errorf("reservation not found")
	}
	if err := s.persistStatus(ctx, id, "confirmed"); err != nil {
		s.mu.Unlock()
		return Record{}, err
	}
	value.Status = "confirmed"
	value.ConfirmedAt = time.Now().Unix()
	s.records[id] = value
	s.mu.Unlock()
	if s.StartProxy != nil {
		if err := s.StartProxy(ctx, value.Identity); err != nil {
			return Record{}, err
		}
	}
	return value, nil
}

// Restore rebuilds confirmed accounts from the durable identity projection.
// TangSeng login is intentionally performed again so no process memory or
// stale token is required after a restart.
func (s *Service) Restore(ctx context.Context) error {
	if s.Store == nil {
		return nil
	}
	identities, err := s.Store.ListIdentities(ctx)
	if err != nil {
		return err
	}
	for _, identity := range identities {
		if identity.Source != domain.SourceTencent || identity.Target != domain.SourceTang || identity.EntityType != "user" {
			continue
		}
		status := "confirmed"
		state, stateErr := s.Store.GetCursor(ctx, "account:"+identity.SourceID)
		if stateErr != nil && !errors.Is(stateErr, store.ErrNotFound) {
			return stateErr
		}
		if state.Aux == "banned" || state.Aux == "tenant_disabled" {
			s.mu.Lock()
			s.records[identity.SourceID] = Record{Identity: identity, Status: state.Aux}
			s.mu.Unlock()
			continue
		}
		if state.Aux == "cancelled" || state.Aux == "deleted" || state.Aux == "registration_pending" {
			continue
		}
		if state.Aux == "deleting" {
			status = "deleting"
		}
		if s.Tang != nil {
			ensured, ensureErr := s.Tang.EnsureUser(ctx, identity)
			if ensureErr != nil {
				var refused *domain.DeliveryError
				if errors.As(ensureErr, &refused) && (refused.Reason == "account_banned" || refused.Reason == "tenant_disabled") {
					state := "banned"
					if refused.Reason == "tenant_disabled" {
						state = "tenant_disabled"
					}
					s.mu.Lock()
					s.records[identity.SourceID] = Record{Identity: identity, Status: state}
					s.mu.Unlock()
					if e := s.Deactivate(ctx, identity.SourceID, state); e != nil {
						return e
					}
					continue
				}
				return fmt.Errorf("restore %s: %w", identity.SourceID, ensureErr)
			}
			identity = ensured
		}
		if err = s.persistLogin(ctx, identity.SourceID); err != nil {
			return err
		}
		record := Record{Identity: identity, Status: status, ConfirmedAt: time.Now().Unix(), LastLoginAt: time.Now().Unix()}
		s.mu.Lock()
		s.records[identity.SourceID] = record
		s.mu.Unlock()
		if s.StartProxy != nil && status == "confirmed" {
			if startErr := s.StartProxy(ctx, identity); startErr != nil {
				return fmt.Errorf("start proxy %s: %w", identity.SourceID, startErr)
			}
		}
	}
	return nil
}
func (s *Service) Cancel(ctx context.Context, id string) error {
	return s.Deactivate(ctx, id, "cancelled")
}

// BeginDeletion also loads a previously cancelled account after a restart.
// Cleanup may log in as the user, but must never start a receiving proxy.
func (s *Service) BeginDeletion(ctx context.Context, id string) (Record, error) {
	r, err := s.Profile(ctx, id)
	if err != nil {
		if s.Store == nil {
			return Record{}, err
		}
		identity, lookup := s.Store.FindIdentity(ctx, domain.SourceTencent, id)
		if lookup != nil {
			return Record{}, lookup
		}
		if identity.Target != domain.SourceTang || identity.EntityType != "user" {
			return Record{}, fmt.Errorf("deletion identity is not a bridged user")
		}
		if err = s.persistStatus(ctx, id, "deleting"); err != nil {
			return Record{}, err
		}
		r = Record{Identity: *identity, Status: "deleting"}
		s.mu.Lock()
		s.records[id] = r
		s.mu.Unlock()
	}
	if err = s.Deactivate(ctx, id, "deleting"); err != nil {
		return Record{}, err
	}
	r.Status = "deleting"
	if s.Tang != nil {
		if s.Tang.UserTokens == nil {
			return Record{}, fmt.Errorf("user credential store unavailable")
		}
		_, hasToken := s.Tang.UserTokens.Load(r.Identity.TargetID)
		if !hasToken {
			if err = s.Refresh(ctx, r.Identity.TargetID); err != nil {
				return Record{}, err
			}
		}
	}
	return r, nil
}
func (s *Service) Deactivate(ctx context.Context, id, status string) error {
	if status != "cancelled" && status != "deleting" && status != "deleted" && status != "banned" && status != "tenant_disabled" {
		return fmt.Errorf("invalid account state")
	}
	if id == "" {
		return fmt.Errorf("account is required")
	}
	if err := s.persistStatus(ctx, id, status); err != nil {
		return err
	}
	s.mu.Lock()
	value, ok := s.records[id]
	if ok {
		value.Status = status
		s.records[id] = value
	}
	s.mu.Unlock()
	if ok {
		if s.StopProxy != nil {
			if err := s.StopProxy(ctx, value.Identity); err != nil {
				return err
			}
		}
		if status != "deleting" && s.Tang != nil && s.Tang.UserTokens != nil {
			s.Tang.UserTokens.Delete(value.Identity.TargetID)
		}
	}
	if status == "deleted" {
		return s.EraseRegistrationCredential(ctx, id)
	}
	return nil
}
func (s *Service) persistStatus(ctx context.Context, id, status string) error {
	if s.Store == nil {
		return nil
	}
	state, err := s.Store.GetCursor(ctx, "account:"+id)
	if errors.Is(err, store.ErrNotFound) {
		state = domain.SyncCursor{Key: "account:" + id, StartedAt: time.Now().Unix()}
	} else if err != nil {
		return err
	}
	if status == "confirmed" && (state.Aux == "deleted" || state.Aux == "deleting") {
		return fmt.Errorf("deleted account cannot be confirmed")
	}
	if status == "confirmed" && (state.Aux == "banned" || state.Aux == "tenant_disabled") {
		return fmt.Errorf("native account requires status verification")
	}
	if (state.Aux == "deleted" && status != "deleted") || (state.Aux == "deleting" && status == "cancelled") {
		return fmt.Errorf("account deletion cannot be reverted")
	}
	state.Aux = status
	state.Value = time.Now().UnixNano()
	return s.Store.PutCursor(ctx, state)
}
func (s *Service) Profile(_ context.Context, id string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.records[id]
	if !ok {
		return Record{}, fmt.Errorf("account not found")
	}
	return value, nil
}
