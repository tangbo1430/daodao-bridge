package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
)

func TestGroupProjectionLiveContract(t *testing.T) {
	raw, err := os.ReadFile("../../test/contract/group-repeated-tips.anonymized.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]json.RawMessage
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture[path.Base(r.URL.Path)])
	}))
	defer server.Close()
	c := TencentClient{BaseURL: server.URL, AdminAccount: "fixture-admin", SecretKey: "fixture-only-secret", SDKAppID: 1}
	profile, err := c.GroupProfile(context.Background(), "dd_fixture-group")
	if err != nil {
		t.Fatal(err)
	}
	if profile.MuteAll != "Off" || profile.Max != 200 {
		t.Fatalf("profile %+v", profile)
	}
	members, err := c.GroupMemberProfiles(context.Background(), "dd_fixture-group")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 || members[1].Role != "Member" || members[1].MutedUntil() != 1790767532 {
		t.Fatalf("members %+v", members)
	}
}
