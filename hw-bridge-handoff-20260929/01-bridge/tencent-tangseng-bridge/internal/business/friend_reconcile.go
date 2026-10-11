package business

import (
	"context"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
	"strconv"
)

// ReconcileFriends starts from a complete native snapshot and only changes
// identities already positively identified as Tang shadows. Pending requests
// and all ordinary Tencent friends are excluded from deletion.
func (s *Service) ReconcileFriends(ctx context.Context, id domain.Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	desired := map[string][2]bool{}
	version := int64(0)
	complete := false
	for page := 0; page < 100; page++ {
		var rows []struct {
			UID         string `json:"uid"`
			Version     int64  `json:"version"`
			Deleted     int    `json:"is_deleted"`
			PeerDeleted int    `json:"be_deleted"`
			Destroyed   int    `json:"is_destroy"`
		}
		if err := s.Tang.UserRequest(ctx, id.TargetID, http.MethodGet, "/v1/friend/sync?api_version=1&limit=200&version="+strconv.FormatInt(version, 10), nil, &rows); err != nil {
			return err
		}
		next := version
		for _, f := range rows {
			if f.UID == "" {
				return fmt.Errorf("friend snapshot uid missing")
			}
			if f.Version > next {
				next = f.Version
			}
			desired[f.UID] = [2]bool{f.Deleted == 0 && f.Destroyed == 0, f.PeerDeleted == 0 && f.Destroyed == 0}
		}
		if len(rows) < 200 {
			complete = true
			break
		}
		if next <= version {
			return fmt.Errorf("native friend cursor stalled")
		}
		version = next
	}
	if !complete {
		return fmt.Errorf("native friend snapshot incomplete")
	}
	actual, err := s.Tencent.FriendIDs(ctx, id.SourceID)
	if err != nil {
		return err
	}
	identities, err := s.Store.ListIdentities(ctx)
	if err != nil {
		return err
	}
	for _, peer := range identities {
		if peer.Source != domain.SourceTang || peer.Target != domain.SourceTencent || peer.SourceID != peer.TargetID || !peer.Shadow || peer.EntityType != "user" || peer.SourceID == "fileHelper" {
			continue
		}
		f, err := s.state(ctx, id.SourceID, peer.SourceID)
		if err != nil {
			return err
		}
		if f.Status == "pending_in" || f.Status == "pending_out" || f.Status == "accepting" || f.Status == "notifying" {
			continue
		}
		want := desired[peer.SourceID]
		if want[0] && want[1] {
			reverse, err := s.Tencent.FriendIDs(ctx, peer.TargetID)
			if err != nil {
				return err
			}
			if !actual[peer.TargetID] || !reverse[id.SourceID] {
				if err = s.Tencent.AddFriend(ctx, id.SourceID, peer.TargetID); err != nil {
					return err
				}
			}
		} else {
			if actual[peer.TargetID] != want[0] {
				if !want[0] {
					f.Status = "deleted"
					if err = s.Store.PutFriendState(ctx, f); err != nil {
						return err
					}
				}
				if err = s.Tencent.SetFriendEdge(ctx, id.SourceID, peer.TargetID, want[0]); err != nil {
					return err
				}
			}
			reverse, err := s.Tencent.FriendIDs(ctx, peer.TargetID)
			if err != nil {
				return err
			}
			if reverse[id.SourceID] != want[1] {
				if err = s.Tencent.SetFriendEdge(ctx, peer.TargetID, id.SourceID, want[1]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
