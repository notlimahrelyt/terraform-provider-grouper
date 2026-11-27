// Package client is a thin, reusable HTTP client for the Grouper Web Services
// REST API. It owns all HTTP and transport concerns (auth, TLS, timeouts,
// status handling, JSON encoding) and is deliberately free of any Terraform
// dependency so it can be unit-tested in isolation with httptest.
//
// NOTE: Grouper endpoint-specific methods (stems, groups, memberships,
// subjects) are intentionally not implemented yet. They are added per phase
// only after the exact request paths, bodies, and response envelopes are
// confirmed against the target Grouper deployment (see docs/api-observations.md).
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// defaultTimeout is used when the caller does not specify one.
	defaultTimeout = 30 * time.Second
	// defaultWSVersion is the Grouper WS API version segment used when the
	// caller does not specify one. Confirmed working against Grouper 4.16.0
	// (see docs/api-observations.md).
	defaultWSVersion = "v2_6_000"
)

// Config holds everything needed to construct a Client. Credentials live only
// here and inside the Authenticator; they are never persisted to Terraform
// state or logged.
type Config struct {
	// BaseURL is the Grouper servicesRest base, e.g.
	// https://grouper.example.edu/grouper-ws/servicesRest
	BaseURL string
	// WSVersion is the Grouper WS API version path segment, e.g. "v2_6_000".
	// It is required by the API and deployment-specific; zero value defaults to
	// defaultWSVersion.
	WSVersion string
	// Auth applies authentication to each request. Required; use NoAuth for
	// unauthenticated/proxy-terminated deployments.
	Auth Authenticator
	// Timeout bounds each request. Zero means defaultTimeout.
	Timeout time.Duration
	// InsecureSkipVerify disables TLS certificate verification. Defaults to
	// false; enabling it is logged as a warning by the provider.
	InsecureSkipVerify bool
	// UserAgent is sent on every request.
	UserAgent string
	// HTTPClient allows tests to inject a custom client/transport. When nil, a
	// client is constructed from Timeout and InsecureSkipVerify.
	HTTPClient *http.Client
}

// Client is the reusable Grouper HTTP client.
type Client struct {
	baseURL    *url.URL
	wsVersion  string
	auth       Authenticator
	userAgent  string
	httpClient *http.Client
}

// New validates the configuration and constructs a Client. It normalizes the
// base URL (stripping a trailing slash) and validates that it uses http or
// https. It does not make any network calls.
func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	normalized := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	u, err := url.Parse(normalized)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("base URL must use http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("base URL must include a host")
	}

	auth := cfg.Auth
	if auth == nil {
		auth = NoAuth{}
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if cfg.InsecureSkipVerify {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in, warned about by provider
		}
		httpClient = &http.Client{
			Timeout:   timeout,
			Transport: transport,
		}
	}

	ua := cfg.UserAgent
	if ua == "" {
		ua = "terraform-provider-grouper"
	}

	wsVersion := strings.TrimSpace(cfg.WSVersion)
	if wsVersion == "" {
		wsVersion = defaultWSVersion
	}

	return &Client{
		baseURL:    u,
		wsVersion:  wsVersion,
		auth:       auth,
		userAgent:  ua,
		httpClient: httpClient,
	}, nil
}

// BaseURL returns the normalized base URL string.
func (c *Client) BaseURL() string { return c.baseURL.String() }

// WSVersion returns the configured Grouper WS API version segment.
func (c *Client) WSVersion() string { return c.wsVersion }

// resolvePath joins the base URL with a relative API path, avoiding malformed
// double slashes.
func (c *Client) resolvePath(p string) string {
	return c.baseURL.String() + "/" + strings.TrimLeft(p, "/")
}

// wsPath builds a version-prefixed API path, e.g. wsPath("stems") ->
// "v2_6_000/stems". Resource path segments (such as a group name embedded in a
// path) must already be URL-encoded by the caller.
func (c *Client) wsPath(p string) string {
	return c.wsVersion + "/" + strings.TrimLeft(p, "/")
}

// doJSON performs a JSON request against the given relative path. If reqBody is
// non-nil it is JSON-encoded as the request body. On a 2xx response, if
// respTarget is non-nil the response body is decoded into it. Non-2xx responses
// are converted to an *APIError wrapping the appropriate sentinel.
//
// This is transport plumbing only: it does not know about any specific Grouper
// resource. Grouper's own result envelope (which can report failure with an
// HTTP 200) is interpreted by the per-resource methods that call this, once the
// envelope shape is confirmed during API discovery.
func (c *Client) doJSON(ctx context.Context, operation, method, path string, reqBody, respTarget any) error {
	var bodyReader io.Reader
	if reqBody != nil {
		encoded, err := json.Marshal(reqBody)
		if err != nil {
			return &APIError{Operation: operation, Method: method, Path: path,
				Message: "failed to encode request body", Err: fmt.Errorf("%w: %v", ErrUnexpectedResponse, err)}
		}
		bodyReader = bytes.NewReader(encoded)
	}

	fullURL := c.resolvePath(path)
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return &APIError{Operation: operation, Method: method, Path: path,
			Message: "failed to build request", Err: fmt.Errorf("%w: %v", ErrUnexpectedResponse, err)}
	}

	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", c.userAgent)
	if err := c.auth.Apply(req); err != nil {
		return &APIError{Operation: operation, Method: method, Path: path,
			Message: "failed to apply authentication", Err: fmt.Errorf("%w: %v", ErrUnexpectedResponse, err)}
	}

	// Log sanitized request metadata only. Never the Authorization header,
	// cookies, or body.
	tflog.Debug(ctx, "grouper request", map[string]any{
		"operation":   operation,
		"method":      method,
		"path":        path,
		"auth_scheme": c.auth.Scheme(),
	})

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Covers context cancellation, timeout, and transport errors.
		return &APIError{Operation: operation, Method: method, Path: path,
			Message: "request failed", Err: fmt.Errorf("%w: %v", ErrUnexpectedResponse, err)}
	}
	defer func() { _ = resp.Body.Close() }()

	duration := time.Since(start)
	tflog.Debug(ctx, "grouper response", map[string]any{
		"operation":   operation,
		"method":      method,
		"path":        path,
		"status":      resp.StatusCode,
		"duration_ms": duration.Milliseconds(),
	})

	// Read the full body once. Grouper always returns a body (result metadata),
	// and it may report an application-level failure via a WsRestResultProblem
	// envelope on either a 2xx or an error status, so the body is inspected
	// before the HTTP status is trusted.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{Operation: operation, Method: method, Path: path, StatusCode: resp.StatusCode,
			Message: "failed to read response body", Err: fmt.Errorf("%w: %v", ErrUnexpectedResponse, err)}
	}

	// A WsRestResultProblem is a definitive, classified failure regardless of
	// HTTP status.
	if probErr := checkResultProblem(operation, body); probErr != nil {
		if apiErr, ok := probErr.(*APIError); ok {
			apiErr.Method = method
			apiErr.Path = path
			apiErr.StatusCode = resp.StatusCode
		}
		return probErr
	}

	if sentinel := classifyStatus(resp.StatusCode); sentinel != nil {
		return &APIError{
			Operation:  operation,
			Method:     method,
			Path:       path,
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
			Err:        sentinel,
		}
	}

	if respTarget != nil {
		if len(bytes.TrimSpace(body)) == 0 {
			return &APIError{Operation: operation, Method: method, Path: path, StatusCode: resp.StatusCode,
				Message: "empty response body", Err: ErrUnexpectedResponse}
		}
		if err := json.Unmarshal(body, respTarget); err != nil {
			return &APIError{Operation: operation, Method: method, Path: path, StatusCode: resp.StatusCode,
				Message: "failed to decode response body", Err: fmt.Errorf("%w: %v", ErrUnexpectedResponse, err)}
		}
	}

	return nil
}
