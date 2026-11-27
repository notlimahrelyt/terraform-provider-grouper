package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type capturedReq struct {
	method      string
	escapedPath string
	body        map[string]any
}

func memberServer(t *testing.T, status int, respBody string, cap *capturedReq) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cap != nil {
			cap.method = r.Method
			cap.escapedPath = r.URL.EscapedPath()
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &cap.body)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, respBody)
	}))
}

func TestAddMembership_SuccessAndRequestShape(t *testing.T) {
	t.Parallel()
	var cap capturedReq
	body := `{"WsAddMemberResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"wsSubject":{"id":"12345","name":"Test User","sourceId":"ldap","resultCode":"SUCCESS","success":"T"}}]}}`
	srv := memberServer(t, http.StatusOK, body, &cap)
	defer srv.Close()

	subj, err := testClient(t, srv).AddMembership(context.Background(),
		"example:apps:myteam:test:admins", SubjectLookup{SubjectID: "abc123"})
	if err != nil {
		t.Fatalf("AddMembership: %v", err)
	}
	if subj.SourceID != "ldap" || subj.ID != "12345" {
		t.Errorf("resolved subject = %+v", subj)
	}

	// PUT to the members path with the group name colon-encoded as %3A.
	if cap.method != http.MethodPut {
		t.Errorf("method = %s, want PUT", cap.method)
	}
	if !strings.HasSuffix(cap.escapedPath, "/groups/example%3Aapps%3Amyteam%3Atest%3Aadmins/members") {
		t.Errorf("escaped path = %s", cap.escapedPath)
	}
	// replaceAllExisting must be "F", and the value dual-filled for id/identifier.
	req := cap.body["WsRestAddMemberRequest"].(map[string]any)
	if req["replaceAllExisting"] != "F" {
		t.Errorf("replaceAllExisting = %v, want F", req["replaceAllExisting"])
	}
	lk := req["subjectLookups"].([]any)[0].(map[string]any)
	if lk["subjectId"] != "abc123" || lk["subjectIdentifier"] != "abc123" {
		t.Errorf("expected dual-filled lookup, got %v", lk)
	}
}

func TestAddMembership_AlreadyExists(t *testing.T) {
	t.Parallel()
	body := `{"WsAddMemberResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"T","resultCode":"SUCCESS_ALREADY_EXISTED"},
	"wsSubject":{"id":"12345","sourceId":"ldap","resultCode":"SUCCESS","success":"T"}}]}}`
	srv := memberServer(t, http.StatusOK, body, nil)
	defer srv.Close()

	if _, err := testClient(t, srv).AddMembership(context.Background(), "example:g", SubjectLookup{SubjectID: "x"}); err != nil {
		t.Errorf("already-existing membership should be success, got %v", err)
	}
}

func TestAddMembership_SubjectNotFound(t *testing.T) {
	t.Parallel()
	body := `{"WsAddMemberResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"F","resultCode":"SUBJECT_NOT_FOUND"},
	"wsSubject":{"id":"nope","resultCode":"SUBJECT_NOT_FOUND","success":"F"}}]}}`
	srv := memberServer(t, http.StatusOK, body, nil)
	defer srv.Close()

	_, err := testClient(t, srv).AddMembership(context.Background(), "example:g", SubjectLookup{SubjectID: "nope"})
	if !IsNotFound(err) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestHasMembership_IsMember(t *testing.T) {
	t.Parallel()
	var cap capturedReq
	body := `{"WsHasMemberResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"T","resultCode":"IS_MEMBER"},
	"wsSubject":{"id":"12345","sourceId":"ldap","resultCode":"SUCCESS","success":"T"}}]}}`
	srv := memberServer(t, http.StatusOK, body, &cap)
	defer srv.Close()

	isMember, subj, err := testClient(t, srv).HasMembership(context.Background(), "example:g", SubjectLookup{SubjectID: "abc123"})
	if err != nil {
		t.Fatalf("HasMembership: %v", err)
	}
	if !isMember {
		t.Error("expected IS_MEMBER -> true")
	}
	if subj.SourceID != "ldap" {
		t.Errorf("sourceId = %q", subj.SourceID)
	}
	// Must request Immediate (direct) membership only, via POST.
	if cap.method != http.MethodPost {
		t.Errorf("method = %s, want POST", cap.method)
	}
	req := cap.body["WsRestHasMemberRequest"].(map[string]any)
	if req["memberFilter"] != "Immediate" {
		t.Errorf("memberFilter = %v, want Immediate", req["memberFilter"])
	}
}

func TestHasMembership_IsNotMember(t *testing.T) {
	t.Parallel()
	body := `{"WsHasMemberResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"T","resultCode":"IS_NOT_MEMBER"},
	"wsSubject":{"id":"12345","sourceId":"ldap","resultCode":"SUCCESS","success":"T"}}]}}`
	srv := memberServer(t, http.StatusOK, body, nil)
	defer srv.Close()

	isMember, _, err := testClient(t, srv).HasMembership(context.Background(), "example:g", SubjectLookup{SubjectID: "x"})
	if err != nil {
		t.Fatalf("HasMembership: %v", err)
	}
	if isMember {
		t.Error("expected IS_NOT_MEMBER -> false")
	}
}

func TestHasMembership_SubjectNotFound(t *testing.T) {
	t.Parallel()
	body := `{"WsHasMemberResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"F","resultCode":"SUBJECT_NOT_FOUND"},
	"wsSubject":{"id":"nope","resultCode":"SUBJECT_NOT_FOUND","success":"F"}}]}}`
	srv := memberServer(t, http.StatusOK, body, nil)
	defer srv.Close()

	_, _, err := testClient(t, srv).HasMembership(context.Background(), "example:g", SubjectLookup{SubjectID: "nope"})
	if !IsNotFound(err) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestDeleteMembership_Success(t *testing.T) {
	t.Parallel()
	body := `{"WsDeleteMemberResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"wsSubject":{"id":"12345","sourceId":"ldap","resultCode":"SUCCESS","success":"T"}}]}}`
	srv := memberServer(t, http.StatusOK, body, nil)
	defer srv.Close()

	if err := testClient(t, srv).DeleteMembership(context.Background(), "example:g", SubjectLookup{SubjectID: "x"}); err != nil {
		t.Fatalf("DeleteMembership: %v", err)
	}
}

func TestDeleteMembership_SubjectNotFoundIsSuccess(t *testing.T) {
	t.Parallel()
	body := `{"WsDeleteMemberResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"F","resultCode":"SUBJECT_NOT_FOUND"},
	"wsSubject":{"id":"nope","resultCode":"SUBJECT_NOT_FOUND","success":"F"}}]}}`
	srv := memberServer(t, http.StatusOK, body, nil)
	defer srv.Close()

	if err := testClient(t, srv).DeleteMembership(context.Background(), "example:g", SubjectLookup{SubjectID: "nope"}); err != nil {
		t.Errorf("subject-not-found on delete should be success, got %v", err)
	}
}

func TestSubjectLookup_DualFill(t *testing.T) {
	t.Parallel()
	// Only id provided -> identifier filled from id.
	if got := (SubjectLookup{SubjectID: "abc"}).toWs(); got.SubjectID != "abc" || got.SubjectIdentifier != "abc" {
		t.Errorf("id-only dual-fill = %+v", got)
	}
	// Only identifier provided -> id filled from identifier.
	if got := (SubjectLookup{SubjectIdentifier: "def"}).toWs(); got.SubjectID != "def" || got.SubjectIdentifier != "def" {
		t.Errorf("identifier-only dual-fill = %+v", got)
	}
	// Source carried through.
	if got := (SubjectLookup{SubjectID: "abc", SubjectSourceID: "ldap"}).toWs(); got.SubjectSourceID != "ldap" {
		t.Errorf("source not carried: %+v", got)
	}
}
