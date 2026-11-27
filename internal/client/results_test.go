package client

import (
	"errors"
	"testing"
)

func TestResultMetadata_Succeeded(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{"T": true, "t": true, " T ": true, "F": false, "": false, "true": false}
	for val, want := range cases {
		m := ResultMetadata{Success: val}
		if got := m.Succeeded(); got != want {
			t.Errorf("Succeeded(%q) = %v, want %v", val, got, want)
		}
	}
}

func TestResultMetadata_AsError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code    string
		success string
		wantErr error
	}{
		{"SUCCESS", "T", nil},
		{"SUCCESS_ALREADY_EXISTED", "T", nil},
		{"INSUFFICIENT_PRIVILEGES", "F", ErrForbidden},
		{"GROUP_NOT_FOUND", "F", ErrNotFound},
		{"STEM_NOT_FOUND", "F", ErrNotFound},
		{"EXCEPTION", "F", ErrUnexpectedResponse},
		{"SOME_UNKNOWN_CODE", "F", ErrUnexpectedResponse},
	}
	for _, tc := range cases {
		err := ResultMetadata{Success: tc.success, ResultCode: tc.code}.AsError("test op")
		if tc.wantErr == nil {
			if err != nil {
				t.Errorf("code %q: expected nil, got %v", tc.code, err)
			}
			continue
		}
		if !errors.Is(err, tc.wantErr) {
			t.Errorf("code %q: got %v, want %v", tc.code, err, tc.wantErr)
		}
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.GrouperResultCode != tc.code {
			t.Errorf("code %q: APIError.GrouperResultCode = %q", tc.code, apiErr.GrouperResultCode)
		}
	}
}

func TestCheckResultProblem(t *testing.T) {
	t.Parallel()

	// The exact envelope observed live against a Grouper 4.16 server.
	liveProblem := []byte(`{"WsRestResultProblem":{"resultMetadata":{"resultCode":"EXCEPTION","resultMessage":"Problem with request","success":"F"},"responseMetadata":{"millis":"1","serverVersion":"4.16.0"}}}`)
	if err := checkResultProblem("probe", liveProblem); !errors.Is(err, ErrUnexpectedResponse) {
		t.Errorf("live problem: got %v, want ErrUnexpectedResponse", err)
	}

	privProblem := []byte(`{"WsRestResultProblem":{"resultMetadata":{"resultCode":"INSUFFICIENT_PRIVILEGES","success":"F"}}}`)
	if err := checkResultProblem("probe", privProblem); !errors.Is(err, ErrForbidden) {
		t.Errorf("privilege problem: got %v, want ErrForbidden", err)
	}

	// A normal success envelope is not a problem.
	normal := []byte(`{"WsFindStemsResults":{"resultMetadata":{"success":"T","resultCode":"SUCCESS"}}}`)
	if err := checkResultProblem("probe", normal); err != nil {
		t.Errorf("normal envelope: got %v, want nil", err)
	}

	// Empty and non-JSON bodies are left for the caller's decode step.
	if err := checkResultProblem("probe", []byte("")); err != nil {
		t.Errorf("empty body: got %v, want nil", err)
	}
	if err := checkResultProblem("probe", []byte("not json")); err != nil {
		t.Errorf("non-json body: got %v, want nil", err)
	}
}
