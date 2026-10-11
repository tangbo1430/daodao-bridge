package business

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type groupFixture struct {
	calls       []call
	members     []domain.GroupMember
	mirrors     []string
	name        string
	capacity    int
	failInvite  bool
	denyUpdate  bool
	creations   int
	groupType   int
	status      int
	failDestroy bool
	forbidden   int
	profile     map[string]any
	roles       map[string]string
	mutes       map[string]int64
}

func groupService(t *testing.T) (*Service, *groupFixture, string, string) {
	t.Helper()
	s, _, owner, peer := fixture(t)
	id, _ := s.owner(context.Background(), owner)
	f := &groupFixture{name: "fixture group", capacity: 200, profile: map[string]any{"MuteAllMember": "Off"}, roles: map[string]string{}, mutes: map[string]int64{}, members: []domain.GroupMember{{UID: id.TargetID, Name: "Owner", Role: 1, Version: 1}, {UID: peer, Name: "Peer", Role: 0, Version: 2}}}
	f.status = 1
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		var raw json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&raw)
		if json.Unmarshal(raw, &body) != nil {
			var members []any
			if json.Unmarshal(raw, &members) == nil {
				body = map[string]any{"members": members}
			}
		}
		f.calls = append(f.calls, call{r.URL.Path, r.Method, r.Header.Get("token"), body})
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/v1/") && r.Header.Get("token") != "owner-fixture-token" {
			http.Error(w, `{"msg":"bad actor"}`, 403)
			return
		}
		var result any = map[string]any{"ActionStatus": "OK", "ErrorCode": 0}
		switch r.URL.Path {
		case "/v1/conversation/sync":
			result = map[string]any{"uid": id.TargetID, "conversations": []any{}}
		case "/v1/friend/sync":
			if r.URL.Query().Get("api_version") != "1" {
				t.Error("missing api_version")
			}
			result = []map[string]any{{"uid": peer, "version": 1}}
		case "/v1/group/create":
			f.creations++
			result = map[string]string{"group_no": "fixture-group"}
		case "/v1/groups/fixture-group":
			if r.Method == "PUT" {
				if f.denyUpdate {
					http.Error(w, `{"msg":"not group owner"}`, 403)
					return
				}
				if n, ok := body["name"].(string); ok {
					f.name = n
				}
			}
			result = domain.GroupState{Number: "fixture-group", Status: f.status, Name: f.name, GroupType: f.groupType, Forbidden: f.forbidden}
		case "/v1/groups/fixture-group/disband":
			if f.denyUpdate {
				http.Error(w, `{"msg":"not owner"}`, 403)
				return
			}
			if f.status == 2 {
				http.Error(w, `{"msg":"already disbanded"}`, 409)
				return
			}
			f.status = 2
		case "/v4/group_open_http_svc/destroy_group":
			if f.failDestroy {
				result = map[string]any{"ActionStatus": "FAIL", "ErrorCode": 10007}
			}
		case "/v1/groups/fixture-group/membersync":
			result = f.members
		case "/v1/groups/fixture-group/members":
			if r.Method == http.MethodGet {
				result = f.members
				break
			}
			if f.failInvite {
				http.Error(w, `{"msg":"transient"}`, 503)
				return
			}
		case "/v4/group_open_http_svc/get_group_info":
			profile := map[string]any{"GroupId": "dd_fixture-group", "MaxMemberNum": f.capacity, "ErrorCode": 0, "Owner_Account": owner}
			for k, v := range f.profile {
				profile[k] = v
			}
			result = map[string]any{"ActionStatus": "OK", "GroupInfo": []map[string]any{profile}}
		case "/v4/group_open_http_svc/create_group":
			for _, key := range []string{"Name", "Notification", "FaceUrl"} {
				if value, ok := body[key]; ok {
					f.profile[key] = value
				}
			}
		case "/v4/group_open_http_svc/get_group_member_info":
			members := []map[string]any{}
			for _, uid := range f.mirrors {
				role := f.roles[uid]
				if role == "" {
					role = "Member"
				}
				if uid == owner {
					role = "Owner"
				}
				members = append(members, map[string]any{"Member_Account": uid, "Role": role, "ShutUpUntil": f.mutes[uid]})
			}
			result = map[string]any{"ActionStatus": "OK", "MemberList": members}
		case "/v4/group_open_http_svc/add_group_member":
			members := []map[string]any{}
			for _, value := range body["MemberList"].([]any) {
				uid := value.(map[string]any)["Member_Account"].(string)
				members = append(members, map[string]any{"Member_Account": uid, "Result": 1})
				f.mirrors = append(f.mirrors, uid)
			}
			result = map[string]any{"ActionStatus": "OK", "MemberList": members}
		case "/v4/group_open_http_svc/modify_group_base_info":
			for k, v := range body {
				if k != "GroupId" {
					f.profile[k] = v
				}
			}
		case "/v4/group_open_http_svc/modify_group_member_info":
			f.roles[body["Member_Account"].(string)] = body["Role"].(string)
		case "/v4/group_open_http_svc/forbid_send_msg":
			for _, uid := range body["Members_Account"].([]any) {
				f.mutes[uid.(string)] = time.Now().Unix() + int64(body["ShutUpTime"].(float64))
			}
		}
		_ = json.NewEncoder(w).Encode(result)
	}))
	t.Cleanup(upstream.Close)
	s.Tang.BaseURL = upstream.URL
	s.Tencent.BaseURL = upstream.URL
	return s, f, owner, peer
}

func TestGroupCreateAndMappingNoDuplicateOnRetry(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	req := GroupRequest{Owner: owner, Members: []string{peer}, Name: "fixture", RequestID: "create-1"}
	for i := 0; i < 2; i++ {
		g, err := s.Group(ctx, "create", req)
		if err != nil {
			t.Fatal(err)
		}
		if !g.MirrorReady || g.TencentID != "dd_fixture-group" || len(g.Members) != 2 {
			t.Fatalf("bad group %+v", g)
		}
	}
	if f.creations != 1 {
		t.Fatalf("created %d times", f.creations)
	}
	mapped, err := s.Store.FindIdentity(ctx, domain.SourceTang, "fixture-group")
	if err != nil || mapped.TargetID != "dd_fixture-group" {
		t.Fatalf("mapping %+v %v", mapped, err)
	}
	for _, c := range f.calls {
		if strings.HasPrefix(c.Path, "/v1/") && c.Token != "owner-fixture-token" {
			t.Fatal("administrator fallback")
		}
		if strings.HasSuffix(c.Path, "change_group_owner") {
			t.Fatal("unchanged owner transfer")
		}
	}
}

func TestSuperGroupNeverCallsUnsupportedMemberSync(t *testing.T) {
	s, f, owner, peer := groupService(t)
	f.groupType = 1
	if _, err := s.Group(context.Background(), "create", GroupRequest{Owner: owner, Members: []string{peer}, RequestID: "super-group"}); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range f.calls {
		if strings.HasSuffix(c.Path, "membersync") {
			t.Fatal("super group used rejected endpoint")
		}
		if strings.HasSuffix(c.Path, "/members") && c.Method == http.MethodGet {
			found = true
		}
	}
	if !found {
		t.Fatal("member pages not fetched")
	}
}
func TestGroupCreateResumesPendingInvitation(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	extra := "nonfriend-fixture"
	_ = s.Store.UpsertIdentity(ctx, domain.Identity{Source: domain.SourceTang, SourceID: extra, Target: domain.SourceTencent, TargetID: extra, EntityType: "user"})
	req := GroupRequest{Owner: owner, Members: []string{peer, extra}, RequestID: "create-resume"}
	f.failInvite = true
	if _, err := s.Group(ctx, "create", req); err == nil {
		t.Fatal("expected invite failure")
	}
	g, _ := s.Store.GetGroupState(ctx, "fixture-group")
	if len(g.PendingInvites) != 1 {
		t.Fatal("lost pending invite")
	}
	f.failInvite = false
	if _, err := s.Group(ctx, "create", req); err != nil {
		t.Fatal(err)
	}
	g, _ = s.Store.GetGroupState(ctx, "fixture-group")
	if f.creations != 1 || len(g.PendingInvites) != 0 {
		t.Fatalf("state %+v creations %d", g, f.creations)
	}
}
func TestGroupCapacityAndPermissionFailuresAreNotSuccess(t *testing.T) {
	s, f, owner, peer := groupService(t)
	ctx := context.Background()
	f.capacity = 1
	f.members[1].Role = 2 // Both members are mandatory and must not be evicted.
	_, err := s.Group(ctx, "create", GroupRequest{Owner: owner, Members: []string{peer}, RequestID: "capacity"})
	if err == nil || !strings.Contains(err.Error(), "容量") {
		t.Fatalf("capacity err %v", err)
	}
	f.capacity = 200
	f.denyUpdate = true
	if _, err = s.Group(ctx, "update", GroupRequest{Owner: owner, GroupID: "dd_fixture-group", Field: "name", Value: "unauthorized"}); err == nil {
		t.Fatal("permission denial hidden")
	}
	if f.name == "unauthorized" {
		t.Fatal("denied name applied")
	}
}
func TestGroupMirrorCallbackAndOrdinaryGroupsAreIgnored(t *testing.T) {
	s, f, owner, _ := groupService(t)
	for _, raw := range []string{fmt.Sprintf(`{"GroupId":"dd_fixture-group","Operator_Account":%q}`, s.Tencent.AdminAccount), fmt.Sprintf(`{"GroupId":"native_group","Operator_Account":%q}`, owner)} {
		if err := s.Handle(context.Background(), domain.BusinessEvent{Source: domain.SourceTencent, Command: "Group.CallbackAfterGroupDestroyed", Payload: json.RawMessage(raw)}); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.calls) != 0 {
		t.Fatal("mirror loop or native group mutation")
	}
}
func TestGroupProjectionRemovesUnacceptedTencentMembers(t *testing.T) {
	s, f, owner, peer := groupService(t)
	f.mirrors = []string{"unauthorized-member"}
	_, err := s.Group(context.Background(), "create", GroupRequest{Owner: owner, Members: []string{peer}, RequestID: "repair"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range f.calls {
		if strings.HasSuffix(c.Path, "delete_group_member") {
			found = true
			items := c.Body["MemberToDel_Account"].([]any)
			if len(items) != 1 || items[0] != "unauthorized-member" {
				t.Fatalf("removed %v", items)
			}
		}
	}
	if !found {
		t.Fatal("rejected projection member remains")
	}
}
