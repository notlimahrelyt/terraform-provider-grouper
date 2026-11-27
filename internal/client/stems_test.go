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

// stemServer spins up a test server that asserts the path/method and returns
// the given status and body.
func stemServer(t *testing.T, wantMethod, wantPathSuffix string, status int, body string, capture *map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != wantMethod {
			t.Errorf("method = %s, want %s", r.Method, wantMethod)
		}
		if !strings.HasSuffix(r.URL.Path, wantPathSuffix) {
			t.Errorf("path = %s, want suffix %s", r.URL.Path, wantPathSuffix)
		}
		if capture != nil {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, capture)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
}

func TestGetStem_Success(t *testing.T) {
	t.Parallel()
	body := `{"WsFindStemsResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"stemResults":[{"name":"example:apps:myteam:example","extension":"example","displayExtension":"example",
	"displayName":"example:apps:myteam:example","description":"Example folder",
	"idIndex":"10001","uuid":"0123456789abcdef0123456789abcdef"}]}}`
	srv := stemServer(t, http.MethodPost, "/v2_6_000/stems", http.StatusOK, body, nil)
	defer srv.Close()

	stem, err := testClient(t, srv).GetStem(context.Background(), StemLookup{Name: "example:apps:myteam:example"})
	if err != nil {
		t.Fatalf("GetStem: %v", err)
	}
	if stem.UUID != "0123456789abcdef0123456789abcdef" {
		t.Errorf("UUID = %q", stem.UUID)
	}
	if stem.Name != "example:apps:myteam:example" || stem.Description != "Example folder" {
		t.Errorf("unexpected stem: %+v", stem)
	}
}

func TestGetStem_ByUUIDUsesUUIDLookup(t *testing.T) {
	t.Parallel()
	var captured map[string]any
	body := `{"WsFindStemsResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"stemResults":[{"name":"example:apps:myteam","uuid":"abc123"}]}}`
	srv := stemServer(t, http.MethodPost, "/stems", http.StatusOK, body, &captured)
	defer srv.Close()

	if _, err := testClient(t, srv).GetStem(context.Background(), StemLookup{UUID: "abc123"}); err != nil {
		t.Fatalf("GetStem: %v", err)
	}
	// Verify the request carried the uuid lookup, not a name.
	req := captured["WsRestFindStemsRequest"].(map[string]any)
	lookups := req["wsStemLookups"].([]any)
	first := lookups[0].(map[string]any)
	if first["uuid"] != "abc123" {
		t.Errorf("expected uuid lookup, got %v", first)
	}
	if _, hasName := first["stemName"]; hasName {
		t.Errorf("uuid lookup should not include stemName: %v", first)
	}
}

func TestGetStem_NotFound(t *testing.T) {
	t.Parallel()
	body := `{"WsFindStemsResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},"stemResults":[]}}`
	srv := stemServer(t, http.MethodPost, "/stems", http.StatusOK, body, nil)
	defer srv.Close()

	_, err := testClient(t, srv).GetStem(context.Background(), StemLookup{Name: "example:nope"})
	if !IsNotFound(err) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestGetStem_RequiresLookup(t *testing.T) {
	t.Parallel()
	c, _ := New(Config{BaseURL: "https://example.edu"})
	if _, err := c.GetStem(context.Background(), StemLookup{}); err == nil {
		t.Error("expected error for empty lookup")
	}
}

func TestSaveStem_Success(t *testing.T) {
	t.Parallel()
	var captured map[string]any
	body := `{"WsStemSaveResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"T","resultCode":"SUCCESS_NO_CHANGES_NEEDED"},
	"wsStem":{"name":"example:test:demo","extension":"demo","displayExtension":"Demo","displayName":"example:test:Demo",
	"description":"a demo","idIndex":"10046","uuid":"ff43c1b175aa438393934896604b75c6"}}]}}`
	srv := stemServer(t, http.MethodPost, "/v2_6_000/stems", http.StatusOK, body, &captured)
	defer srv.Close()

	stem, err := testClient(t, srv).SaveStem(context.Background(), SaveStemRequest{
		Name: "example:test:demo", DisplayName: "Demo", Description: "a demo",
	})
	if err != nil {
		t.Fatalf("SaveStem: %v", err)
	}
	if stem.UUID != "ff43c1b175aa438393934896604b75c6" {
		t.Errorf("UUID = %q", stem.UUID)
	}
	// Verify the request derived the extension from the full name.
	req := captured["WsRestStemSaveRequest"].(map[string]any)
	saves := req["wsStemToSaves"].([]any)
	ws := saves[0].(map[string]any)["wsStem"].(map[string]any)
	if ws["extension"] != "demo" {
		t.Errorf("extension = %v, want demo", ws["extension"])
	}
}

func TestSaveStem_ForbiddenResultCode(t *testing.T) {
	t.Parallel()
	// Top-level failure with an INSUFFICIENT_PRIVILEGES code on HTTP 200.
	body := `{"WsStemSaveResults":{"resultMetadata":{"success":"F","resultCode":"INSUFFICIENT_PRIVILEGES"}}}`
	srv := stemServer(t, http.MethodPost, "/stems", http.StatusOK, body, nil)
	defer srv.Close()

	_, err := testClient(t, srv).SaveStem(context.Background(), SaveStemRequest{Name: "example:test:demo"})
	if !IsForbidden(err) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

func TestDeleteStem_Success(t *testing.T) {
	t.Parallel()
	body := `{"WsStemDeleteResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"T","resultCode":"SUCCESS"}}]}}`
	srv := stemServer(t, http.MethodPost, "/v2_6_000/stems", http.StatusOK, body, nil)
	defer srv.Close()

	if err := testClient(t, srv).DeleteStem(context.Background(), StemLookup{Name: "example:test:demo"}); err != nil {
		t.Fatalf("DeleteStem: %v", err)
	}
}

func TestDeleteStem_AlreadyGoneIsSuccess(t *testing.T) {
	t.Parallel()
	body := `{"WsStemDeleteResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"},
	"results":[{"resultMetadata":{"success":"T","resultCode":"SUCCESS_STEM_NOT_FOUND"}}]}}`
	srv := stemServer(t, http.MethodPost, "/stems", http.StatusOK, body, nil)
	defer srv.Close()

	if err := testClient(t, srv).DeleteStem(context.Background(), StemLookup{Name: "example:test:gone"}); err != nil {
		t.Errorf("already-absent stem should be success, got %v", err)
	}
}

func TestLastPathComponent(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"example:apps:myteam:azure": "azure",
		"example":                   "example",
		"a:b":                       "b",
	}
	for in, want := range cases {
		if got := lastPathComponent(in); got != want {
			t.Errorf("lastPathComponent(%q) = %q, want %q", in, got, want)
		}
	}
}
