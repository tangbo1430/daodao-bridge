package business

import (
	"context"
	"strings"
	"testing"
)

func TestGroupDisbandResumesNativeSuccessAndIsIdempotent(t *testing.T) {
	s, f, owner, _ := groupService(t)
	ctx := context.Background()
	req := GroupRequest{Owner: owner, GroupID: "dd_fixture-group"}
	if _, err := s.Group(ctx, "sync", req); err != nil {
		t.Fatal(err)
	}
	f.failDestroy = true
	if _, err := s.Group(ctx, "disband", req); err == nil {
		t.Fatal("mirror failure hidden")
	}
	if f.status != 2 {
		t.Fatal("native disband did not succeed")
	}
	f.failDestroy = false
	if g, err := s.Group(ctx, "disband", req); err != nil || g.Status != 2 || g.MirrorReady {
		t.Fatalf("resume %+v %v", g, err)
	}
	n := len(f.calls)
	if _, err := s.Group(ctx, "disband", req); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != n {
		t.Fatal("completed disband made remote calls")
	}
	native := 0
	for _, c := range f.calls {
		if strings.HasSuffix(c.Path, "/disband") {
			native++
		}
	}
	if native != 1 {
		t.Fatalf("native disband repeated %d", native)
	}
}

func TestDisbandPermissionDenialDoesNotDestroyMirror(t *testing.T) {
	s, f, owner, _ := groupService(t)
	req := GroupRequest{Owner: owner, GroupID: "dd_fixture-group"}
	if _, err := s.Group(context.Background(), "sync", req); err != nil {
		t.Fatal(err)
	}
	f.calls = nil
	f.denyUpdate = true
	if _, err := s.Group(context.Background(), "disband", req); err == nil {
		t.Fatal("permission denied reported success")
	}
	for _, c := range f.calls {
		if strings.HasSuffix(c.Path, "destroy_group") {
			t.Fatal("destroyed mirror after native denial")
		}
	}
}
