package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew_NormalizesBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		want    string
		wantErr bool
	}{
		{name: "trailing slash trimmed", baseURL: "https://g.example.edu/grouper-ws/servicesRest/", want: "https://g.example.edu/grouper-ws/servicesRest"},
		{name: "multiple trailing slashes trimmed", baseURL: "https://g.example.edu/x///", want: "https://g.example.edu/x"},
		{name: "no trailing slash preserved", baseURL: "https://g.example.edu/x", want: "https://g.example.edu/x"},
		{name: "surrounding whitespace trimmed", baseURL: "  https://g.example.edu/x  ", want: "https://g.example.edu/x"},
		{name: "http allowed", baseURL: "http://g.example.edu/x", want: "http://g.example.edu/x"},
		{name: "empty rejected", baseURL: "", wantErr: true},
		{name: "non-http scheme rejected", baseURL: "ftp://g.example.edu/x", wantErr: true},
		{name: "missing host rejected", baseURL: "https:///nohost", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, err := New(Config{BaseURL: tc.baseURL})
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := c.BaseURL(); got != tc.want {
				t.Errorf("BaseURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestClient_resolvePath_NoDoubleSlash(t *testing.T) {
	t.Parallel()
	c, err := New(Config{BaseURL: "https://g.example.edu/grouper-ws/servicesRest/"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	want := "https://g.example.edu/grouper-ws/servicesRest/v4_0_000/stems"
	if got := c.resolvePath("/v4_0_000/stems"); got != want {
		t.Errorf("resolvePath leading slash = %q, want %q", got, want)
	}
	if got := c.resolvePath("v4_0_000/stems"); got != want {
		t.Errorf("resolvePath no leading slash = %q, want %q", got, want)
	}
}

func TestClassifyStatus(t *testing.T) {
	t.Parallel()
	cases := map[int]error{
		200: nil,
		201: nil,
		204: nil,
		401: ErrUnauthorized,
		403: ErrForbidden,
		404: ErrNotFound,
		409: ErrConflict,
		500: ErrUnexpectedResponse,
		418: ErrUnexpectedResponse,
	}
	for status, want := range cases {
		if got := classifyStatus(status); !errors.Is(got, want) {
			t.Errorf("classifyStatus(%d) = %v, want %v", status, got, want)
		}
	}
}

// testClient builds a client pointed at the given test server.
func testClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := New(Config{BaseURL: srv.URL, HTTPClient: srv.Client(), Auth: NoAuth{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestDoJSON_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Errorf("missing User-Agent")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":"ok"}`))
	}))
	defer srv.Close()

	var out struct {
		Value string `json:"value"`
	}
	if err := testClient(t, srv).doJSON(context.Background(), "test", http.MethodGet, "/thing", nil, &out); err != nil {
		t.Fatalf("doJSON: %v", err)
	}
	if out.Value != "ok" {
		t.Errorf("decoded value = %q, want ok", out.Value)
	}
}

func TestDoJSON_SendsBodyAndContentType(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "example:test" {
			t.Errorf("body name = %q", body["name"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	req := map[string]string{"name": "example:test"}
	var out map[string]any
	if err := testClient(t, srv).doJSON(context.Background(), "test", http.MethodPost, "/thing", req, &out); err != nil {
		t.Fatalf("doJSON: %v", err)
	}
}

func TestDoJSON_StatusToSentinel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrForbidden},
		{http.StatusNotFound, ErrNotFound},
		{http.StatusConflict, ErrConflict},
		{http.StatusInternalServerError, ErrUnexpectedResponse},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
		}))
		err := testClient(t, srv).doJSON(context.Background(), "test", http.MethodGet, "/thing", nil, nil)
		srv.Close()
		if !errors.Is(err, tc.want) {
			t.Errorf("status %d: got %v, want %v", tc.status, err, tc.want)
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Errorf("status %d: expected *APIError", tc.status)
		} else if apiErr.StatusCode != tc.status {
			t.Errorf("APIError.StatusCode = %d, want %d", apiErr.StatusCode, tc.status)
		}
	}
}

func TestDoJSON_MalformedJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer srv.Close()

	var out map[string]any
	err := testClient(t, srv).doJSON(context.Background(), "test", http.MethodGet, "/thing", nil, &out)
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Errorf("got %v, want ErrUnexpectedResponse", err)
	}
}

func TestDoJSON_EmptyBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var out map[string]any
	err := testClient(t, srv).doJSON(context.Background(), "test", http.MethodGet, "/thing", nil, &out)
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Errorf("got %v, want ErrUnexpectedResponse", err)
	}
}

func TestDoJSON_ContextCancellation(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := testClient(t, srv).doJSON(ctx, "test", http.MethodGet, "/thing", nil, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Errorf("got %v, want ErrUnexpectedResponse", err)
	}
}

func TestBasicAuth_SetsHeaderNotLogged(t *testing.T) {
	t.Parallel()
	var gotUser, gotPass string
	var hadAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, hadAuth = r.BasicAuth()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := New(Config{BaseURL: srv.URL, HTTPClient: srv.Client(), Auth: BasicAuth{Username: "svc", Password: "secret"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var out map[string]any
	if err := c.doJSON(context.Background(), "test", http.MethodGet, "/thing", nil, &out); err != nil {
		t.Fatalf("doJSON: %v", err)
	}
	if !hadAuth || gotUser != "svc" || gotPass != "secret" {
		t.Errorf("basic auth not applied: user=%q hadAuth=%v", gotUser, hadAuth)
	}
	if (BasicAuth{Username: "svc", Password: "secret"}).Scheme() != "basic" {
		t.Errorf("unexpected scheme label")
	}
}
