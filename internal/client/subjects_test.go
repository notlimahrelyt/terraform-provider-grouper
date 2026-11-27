package client

import (
	"context"
	"net/http"
	"testing"
)

func TestGetSubject_Success(t *testing.T) {
	t.Parallel()
	var captured map[string]any
	// Mirrors a live Grouper 4.16 response shape (dual-match, identifier resolution).
	body := `{"WsGetSubjectsResults":{"subjectAttributeNames":["name"],
	"wsSubjects":[{"identifierLookup":"abc123","resultCode":"SUCCESS","success":"T","id":"12345678",
	"name":"Example User","sourceId":"sourceId","attributeValues":["Example User"]}],
	"resultMetadata":{"resultCode":"SUCCESS","resultMessage":"Queried 1 subjects","success":"T"}}}`
	srv := stemServer(t, http.MethodPost, "/v2_6_000/subjects", http.StatusOK, body, &captured)
	defer srv.Close()

	subj, err := testClient(t, srv).GetSubject(context.Background(), SubjectLookup{SubjectID: "abc123"}, []string{"name"})
	if err != nil {
		t.Fatalf("GetSubject: %v", err)
	}
	if subj.ID != "12345678" || subj.Name != "Example User" || subj.SourceID != "sourceId" {
		t.Errorf("unexpected subject: %+v", subj)
	}
	// dual-match + only the name attribute requested.
	req := captured["WsRestGetSubjectsRequest"].(map[string]any)
	lk := req["wsSubjectLookups"].([]any)[0].(map[string]any)
	if lk["subjectId"] != "abc123" || lk["subjectIdentifier"] != "abc123" {
		t.Errorf("expected dual-filled lookup, got %v", lk)
	}
	attrs := req["subjectAttributeNames"].([]any)
	if len(attrs) != 1 || attrs[0] != "name" {
		t.Errorf("expected only [name], got %v", attrs)
	}
}

func TestGetSubject_Attributes(t *testing.T) {
	t.Parallel()
	var captured map[string]any
	// attributeValues are positional against the response's subjectAttributeNames.
	body := `{"WsGetSubjectsResults":{"subjectAttributeNames":["emailaddress","netid","lastname"],
	"wsSubjects":[{"resultCode":"SUCCESS","success":"T","id":"14820095","name":"Example User","sourceId":"sourceId",
	"attributeValues":["user@example.edu","abc123","User"]}],
	"resultMetadata":{"resultCode":"SUCCESS","success":"T"}}}`
	srv := stemServer(t, http.MethodPost, "/subjects", http.StatusOK, body, &captured)
	defer srv.Close()

	subj, err := testClient(t, srv).GetSubject(context.Background(),
		SubjectLookup{SubjectID: "abc123"}, []string{"emailaddress", "netid", "lastname"})
	if err != nil {
		t.Fatalf("GetSubject: %v", err)
	}
	if subj.Attributes["emailaddress"] != "user@example.edu" {
		t.Errorf("emailaddress = %q", subj.Attributes["emailaddress"])
	}
	if subj.Attributes["netid"] != "abc123" || subj.Attributes["lastname"] != "User" {
		t.Errorf("unexpected attributes: %v", subj.Attributes)
	}
	// The requested names must be sent through.
	req := captured["WsRestGetSubjectsRequest"].(map[string]any)
	names := req["subjectAttributeNames"].([]any)
	if len(names) != 3 || names[0] != "emailaddress" {
		t.Errorf("requested names = %v", names)
	}
}

func TestGetSubject_NotFoundEmpty(t *testing.T) {
	t.Parallel()
	body := `{"WsGetSubjectsResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},"wsSubjects":[]}}`
	srv := stemServer(t, http.MethodPost, "/subjects", http.StatusOK, body, nil)
	defer srv.Close()

	_, err := testClient(t, srv).GetSubject(context.Background(), SubjectLookup{SubjectID: "nobody"}, nil)
	if !IsNotFound(err) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestGetSubject_NotFoundResultCode(t *testing.T) {
	t.Parallel()
	body := `{"WsGetSubjectsResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"wsSubjects":[{"id":"nobody","resultCode":"SUBJECT_NOT_FOUND","success":"F"}]}}`
	srv := stemServer(t, http.MethodPost, "/subjects", http.StatusOK, body, nil)
	defer srv.Close()

	_, err := testClient(t, srv).GetSubject(context.Background(), SubjectLookup{SubjectID: "nobody"}, nil)
	if !IsNotFound(err) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestGetSubject_RequiresInput(t *testing.T) {
	t.Parallel()
	c, _ := New(Config{BaseURL: "https://example.edu"})
	if _, err := c.GetSubject(context.Background(), SubjectLookup{}, nil); err == nil {
		t.Error("expected error for empty lookup")
	}
}
