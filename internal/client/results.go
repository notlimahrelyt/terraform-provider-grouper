package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

// This file models the Grouper Web Services response envelope, which is shared
// by every operation. The shapes here are taken verbatim from confirmed
// responses (see docs/api-observations.md) — an error envelope observed from a live Grouper 4.16 server plus
// the upstream request/response samples. Per-operation result wrappers
// (WsStemSaveResults, WsFindGroupsResults, ...) embed ResultMetadata and
// ResponseMetadata and are defined alongside the methods that use them, added
// phase by phase.

// ResultMetadata is the per-response (and per-item) status block Grouper
// returns. Numeric-looking fields are strings in the JSON, matching Grouper.
type ResultMetadata struct {
	Success       string `json:"success"`
	ResultCode    string `json:"resultCode"`
	ResultMessage string `json:"resultMessage"`
}

// ResponseMetadata carries non-status response info, notably the server version.
type ResponseMetadata struct {
	Millis        string `json:"millis"`
	ServerVersion string `json:"serverVersion"`
}

// Succeeded reports whether Grouper flagged this result as successful
// (success == "T", case-insensitive). Grouper can return an application-level
// failure with HTTP 200, so callers must consult this, not the HTTP status.
func (m ResultMetadata) Succeeded() bool {
	return strings.EqualFold(strings.TrimSpace(m.Success), "T")
}

// AsError returns nil when the result succeeded, otherwise an *APIError whose
// wrapped sentinel is classified from the Grouper result code. The full
// resultMessage is not embedded verbatim (it can be large and echo request
// details); only the sanitized code is surfaced.
func (m ResultMetadata) AsError(operation string) error {
	if m.Succeeded() {
		return nil
	}
	return &APIError{
		Operation:         operation,
		GrouperResultCode: m.ResultCode,
		Message:           "Grouper reported a non-success result",
		Err:               sentinelForResultCode(m.ResultCode),
	}
}

// sentinelForResultCode maps a Grouper application-level result code to a
// sentinel error. Only substrings observed in real responses are matched; any
// unrecognized failure code falls through to ErrUnexpectedResponse rather than
// being guessed at.
func sentinelForResultCode(code string) error {
	upper := strings.ToUpper(code)
	switch {
	case strings.Contains(upper, "INSUFFICIENT_PRIVILEGES"):
		return ErrForbidden
	case strings.Contains(upper, "NOT_FOUND"):
		return ErrNotFound
	default:
		return ErrUnexpectedResponse
	}
}

// wsRestResultProblemEnvelope matches the top-level error Grouper returns when a
// request never reaches an operation handler (confirmed live against Grouper 4.16).
type wsRestResultProblemEnvelope struct {
	Problem *struct {
		ResultMetadata   ResultMetadata   `json:"resultMetadata"`
		ResponseMetadata ResponseMetadata `json:"responseMetadata"`
	} `json:"WsRestResultProblem"`
}

// checkResultProblem inspects a raw 2xx response body for a top-level
// WsRestResultProblem envelope and returns a classified *APIError if present.
// It returns nil when the body is not a problem envelope, leaving normal
// decoding to the caller. Unparseable bodies are left for the caller's own
// decode step to report, so this never masks a decode error.
func checkResultProblem(operation string, body []byte) error {
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}
	var env wsRestResultProblemEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil
	}
	if env.Problem == nil {
		return nil
	}
	meta := env.Problem.ResultMetadata
	return &APIError{
		Operation:         operation,
		GrouperResultCode: meta.ResultCode,
		Message:           fmt.Sprintf("Grouper rejected the request (server %s)", env.Problem.ResponseMetadata.ServerVersion),
		Err:               sentinelForResultCode(meta.ResultCode),
	}
}
