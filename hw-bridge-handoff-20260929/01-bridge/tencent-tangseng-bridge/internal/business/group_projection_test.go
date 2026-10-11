package business

import (
	"context"
	"strings"
	"testing"
	"time"
)

func projectionWrites(calls []call) int {
	n := 0
	for _, c := range calls {
		if strings.Contains(c.Path, "/group_open_http_svc/") && !strings.Contains(c.Path, "/get_group_") {
			n++
		}
	}
	return n
}

func TestUnchangedGroupSyncDoesNotEmitSystemTips(t *testing.T) {
	s, f, owner, _ := groupService(t)
	req := GroupRequest{Owner: owner, GroupID: "dd_fixture-group"}
	if _, err := s.Group(context.Background(), "sync", req); err != nil {
		t.Fatal(err)
	}
	// The real Tencent response retains a past cancellation timestamp.
	for _, uid := range f.mirrors {
		f.mutes[uid] = time.Now().Unix() - 1
	}
	start := len(f.calls)
	for i := 0; i < 3; i++ {
		if _, err := s.Group(context.Background(), "sync", req); err != nil {
			t.Fatal(err)
		}
	}
	if n := projectionWrites(f.calls[start:]); n != 0 {
		t.Fatalf("unchanged sync emitted %d writes", n)
	}
}

func TestChangedMuteAndRoleAppliedOnlyOnce(t *testing.T) {
	s, f, owner, peer := groupService(t)
	req := GroupRequest{Owner: owner, GroupID: "dd_fixture-group"}
	sync := func() {
		t.Helper()
		if _, err := s.Group(context.Background(), "sync", req); err != nil {
			t.Fatal(err)
		}
	}
	sync()
	f.forbidden = 1
	f.members[1].Role = 2
	f.members[1].MutedUntil = time.Now().Unix() + 600
	start := len(f.calls)
	sync()
	if n := projectionWrites(f.calls[start:]); n != 3 {
		t.Fatalf("wanted 3 changed writes, got %d", n)
	}
	if f.profile["MuteAllMember"] != "On" || f.roles[peer] != "Admin" || f.mutes[peer] <= time.Now().Unix() {
		t.Fatal("mute/role not applied")
	}
	start = len(f.calls)
	sync()
	if n := projectionWrites(f.calls[start:]); n != 0 {
		t.Fatalf("mute repeated: %d", n)
	}
	f.forbidden = 0
	f.members[1].Role = 0
	f.members[1].MutedUntil = 0
	start = len(f.calls)
	sync()
	if n := projectionWrites(f.calls[start:]); n != 3 {
		t.Fatalf("wanted 3 inverse writes, got %d", n)
	}
	start = len(f.calls)
	sync()
	if n := projectionWrites(f.calls[start:]); n != 0 {
		t.Fatalf("unmute repeated: %d", n)
	}
	// Repair genuine Tencent drift even if the persisted Tang snapshot matches.
	f.profile["MuteAllMember"] = "On"
	f.mutes[peer] = time.Now().Unix() + 600
	start = len(f.calls)
	sync()
	if n := projectionWrites(f.calls[start:]); n != 2 {
		t.Fatalf("drift not repaired: %d", n)
	}
}

func TestMemberMuteTimestampEquivalence(t *testing.T) {
	for _, tc := range []struct {
		seconds, current int64
		changed          bool
	}{
		{0, 0, false}, {0, 999, false}, {0, 1000, false}, {0, 1001, true},
		{60, 0, true}, {60, 1060, false}, {60, 1063, false}, {60, 1070, true},
	} {
		if got := memberMuteChanged(1000, tc.seconds, tc.current); got != tc.changed {
			t.Errorf("%+v got %v", tc, got)
		}
	}
}
