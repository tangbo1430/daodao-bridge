package business

import (
	"context"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
	"net/url"
	"time"
)

type peerSetting struct {
	Receipt bool
	Expires int64
}

func (s *Service) Prepare(ctx context.Context, e domain.MessageEvent, from, to domain.Identity) (domain.MessageEvent, error) {
	if e.Source == domain.SourceTencent {
		if _, err := s.owner(ctx, e.FromAccount); err != nil {
			return e, err
		}
	}
	if e.Source == domain.SourceTencent && e.ConversationType == domain.ConversationC2C {
		key := from.TargetID + ":" + to.TargetID
		cached, ok := s.peerSettings.Load(key)
		setting, _ := cached.(peerSetting)
		if !ok || setting.Expires < time.Now().Unix() {
			var profile struct {
				Receipt int `json:"receipt"`
			}
			if err := s.Tang.UserRequest(ctx, from.TargetID, http.MethodGet, "/v1/users/"+url.PathEscape(to.TargetID), nil, &profile); err != nil {
				return e, err
			}
			setting = peerSetting{Receipt: profile.Receipt == 1, Expires: time.Now().Add(time.Minute).Unix()}
			s.peerSettings.Store(key, setting)
		}
		e.NeedReceipt = setting.Receipt
	}
	if e.ConversationType == domain.ConversationGroup {
		e.NeedReceipt = false
	}
	var err error
	e, err = s.prepareMetadata(ctx, e, from, to)
	if err != nil {
		return e, err
	}
	if s.Media != nil {
		return s.Media.Prepare(ctx, e, from, to)
	}
	return e, nil
}
