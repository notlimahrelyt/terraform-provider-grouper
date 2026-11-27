package client

import (
	"context"
	"net/http"
	"testing"
)

func TestGetGroup_Success(t *testing.T) {
	t.Parallel()
	var captured map[string]any
	body := `{"WsFindGroupsResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"groupResults":[{"name":"example:apps:myteam:test:admins","extension":"admins","displayExtension":"Admins",
	"displayName":"example:apps:myteam:Terraform Test:Admins","description":"admins group","enabled":"T",
	"typeOfGroup":"group","idIndex":"10009","uuid":"32ca90db41b04a1a9611a214a20bac42"}]}}`
	srv := stemServer(t, http.MethodPost, "/v2_6_000/groups", http.StatusOK, body, &captured)
	defer srv.Close()

	g, err := testClient(t, srv).GetGroup(context.Background(), GroupLookup{Name: "example:apps:myteam:test:admins"})
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if g.UUID != "32ca90db41b04a1a9611a214a20bac42" || !g.Enabled || g.TypeOfGroup != "group" {
		t.Errorf("unexpected group: %+v", g)
	}
	// Verify a direct name lookup was used (not a query filter).
	req := captured["WsRestFindGroupsRequest"].(map[string]any)
	lk := req["wsGroupLookups"].([]any)[0].(map[string]any)
	if lk["groupName"] != "example:apps:myteam:test:admins" {
		t.Errorf("expected groupName lookup, got %v", lk)
	}
}

func TestGetGroup_ByUUID(t *testing.T) {
	t.Parallel()
	var captured map[string]any
	body := `{"WsFindGroupsResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"groupResults":[{"name":"example:g","uuid":"abc"}]}}`
	srv := stemServer(t, http.MethodPost, "/groups", http.StatusOK, body, &captured)
	defer srv.Close()

	if _, err := testClient(t, srv).GetGroup(context.Background(), GroupLookup{UUID: "abc"}); err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	lk := captured["WsRestFindGroupsRequest"].(map[string]any)["wsGroupLookups"].([]any)[0].(map[string]any)
	if lk["uuid"] != "abc" {
		t.Errorf("expected uuid lookup, got %v", lk)
	}
	if _, hasName := lk["groupName"]; hasName {
		t.Errorf("uuid lookup should not include groupName: %v", lk)
	}
}

func TestGetGroup_NotFound(t *testing.T) {
	t.Parallel()
	body := `{"WsFindGroupsResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},"groupResults":[]}}`
	srv := stemServer(t, http.MethodPost, "/groups", http.StatusOK, body, nil)
	defer srv.Close()

	_, err := testClient(t, srv).GetGroup(context.Background(), GroupLookup{Name: "example:nope"})
	if !IsNotFound(err) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestSaveGroup_Success(t *testing.T) {
	t.Parallel()
	var captured map[string]any
	body := `{"WsGroupSaveResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"T","resultCode":"SUCCESS_INSERTED"},
	"wsGroup":{"name":"example:apps:myteam:test:admins","extension":"admins","displayExtension":"Admins",
	"description":"admins","enabled":"T","typeOfGroup":"group","idIndex":"10050","uuid":"09b488181c7d42ba9eb880fd7ad698d3"}}]}}`
	srv := stemServer(t, http.MethodPost, "/v2_6_000/groups", http.StatusOK, body, &captured)
	defer srv.Close()

	g, err := testClient(t, srv).SaveGroup(context.Background(), SaveGroupRequest{
		Name: "example:apps:myteam:test:admins", DisplayName: "Admins", Description: "admins",
	})
	if err != nil {
		t.Fatalf("SaveGroup: %v", err)
	}
	if g.UUID != "09b488181c7d42ba9eb880fd7ad698d3" {
		t.Errorf("UUID = %q", g.UUID)
	}
	ws := captured["WsRestGroupSaveRequest"].(map[string]any)["wsGroupToSaves"].([]any)[0].(map[string]any)["wsGroup"].(map[string]any)
	if ws["extension"] != "admins" {
		t.Errorf("extension = %v, want admins", ws["extension"])
	}
}

func TestDeleteGroup_AlreadyGoneIsSuccess(t *testing.T) {
	t.Parallel()
	body := `{"WsGroupDeleteResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"T","resultCode":"SUCCESS_GROUP_NOT_FOUND"}}]}}`
	srv := stemServer(t, http.MethodPost, "/groups", http.StatusOK, body, nil)
	defer srv.Close()

	if err := testClient(t, srv).DeleteGroup(context.Background(), GroupLookup{Name: "example:gone"}); err != nil {
		t.Errorf("already-absent group should be success, got %v", err)
	}
}

func TestDeleteGroup_RequiresName(t *testing.T) {
	t.Parallel()
	c, _ := New(Config{BaseURL: "https://example.edu"})
	if err := c.DeleteGroup(context.Background(), GroupLookup{}); err == nil {
		t.Error("expected error for missing name")
	}
}
