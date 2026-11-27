package client

import "net/http"

// Authenticator applies authentication to an outgoing request. It is an
// interface so the institutional Grouper deployment can use something other
// than standard HTTP Basic auth (for example, a bearer token or a custom
// header) without changing the rest of the client.
//
// Implementations must never log or otherwise expose the credentials they set.
type Authenticator interface {
	// Apply mutates the request to add whatever authentication the deployment
	// requires (headers, etc.). It runs on every request.
	Apply(req *http.Request) error
	// Scheme returns a short, non-sensitive label for logging/diagnostics,
	// e.g. "basic" or "none". It must never contain secret material.
	Scheme() string
}

// BasicAuth implements Authenticator using HTTP Basic authentication, the most
// common Grouper Web Services configuration. The password is held in memory
// only and is never logged.
type BasicAuth struct {
	Username string
	Password string
}

// Apply sets the Authorization header for HTTP Basic auth.
func (a BasicAuth) Apply(req *http.Request) error {
	req.SetBasicAuth(a.Username, a.Password)
	return nil
}

// Scheme returns the non-sensitive scheme label.
func (a BasicAuth) Scheme() string { return "basic" }

// NoAuth performs no authentication. It is primarily useful for tests against a
// local mock server and for deployments that terminate auth at a proxy.
type NoAuth struct{}

// Apply is a no-op.
func (NoAuth) Apply(*http.Request) error { return nil }

// Scheme returns "none".
func (NoAuth) Scheme() string { return "none" }

// Ensure the concrete types satisfy the interface at compile time.
var (
	_ Authenticator = BasicAuth{}
	_ Authenticator = NoAuth{}
)
