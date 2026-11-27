// Package provider implements the Terraform provider for Internet2 Grouper
// using the Terraform Plugin Framework.
package provider

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/notlimahrelyt/terraform-provider-grouper/internal/client"
)

// Environment variables used as fallbacks for provider configuration.
const (
	envBaseURL = "GROUPER_BASE_URL"
	envUser    = "GROUPER_USERNAME"
	envPass    = "GROUPER_PASSWORD"
	envTimeout = "GROUPER_REQUEST_TIMEOUT"
)

// Ensure grouperProvider satisfies the framework interface.
var _ provider.Provider = (*grouperProvider)(nil)

// grouperProvider is the provider implementation.
type grouperProvider struct {
	// version is set at build time and surfaced in the User-Agent.
	version string
}

// grouperProviderModel maps provider schema attributes to Go values.
type grouperProviderModel struct {
	BaseURL            types.String `tfsdk:"base_url"`
	WSVersion          types.String `tfsdk:"ws_version"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	RequestTimeout     types.String `tfsdk:"request_timeout"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
}

// New returns a function that constructs the provider, as required by
// providerserver.Serve.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &grouperProvider{version: version}
	}
}

func (p *grouperProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "grouper"
	resp.Version = p.version
}

func (p *grouperProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Internet2 Grouper folders/stems, groups, and direct memberships " +
			"through the Grouper Web Services REST API.",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional: true,
				Description: "Base URL of the Grouper Web Services REST endpoint, e.g. " +
					"\"https://grouper.example.edu/grouper-ws/servicesRest\". " +
					"May also be set with the " + envBaseURL + " environment variable.",
			},
			"ws_version": schema.StringAttribute{
				Optional: true,
				Description: "Grouper Web Services API version path segment (e.g. \"v2_6_000\"). " +
					"Defaults to \"v2_6_000\", which is confirmed against Grouper 4.16. " +
					"Set this if your deployment requires a different version segment.",
			},
			"username": schema.StringAttribute{
				Optional: true,
				Description: "Username for the Grouper service account. " +
					"May also be set with the " + envUser + " environment variable.",
			},
			"password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Password for the Grouper service account. " +
					"May also be set with the " + envPass + " environment variable. " +
					"Prefer the environment variable so the secret does not appear in configuration.",
			},
			"request_timeout": schema.StringAttribute{
				Optional: true,
				Description: "Per-request timeout as a Go duration string (e.g. \"30s\", \"1m\"). " +
					"Defaults to \"30s\". May also be set with the " + envTimeout + " environment variable.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional: true,
				Description: "Disable TLS certificate verification. Defaults to false. " +
					"Enabling this is insecure and should only be used for testing against " +
					"self-signed development endpoints.",
			},
		},
	}
}

// Configure reads provider configuration (with environment-variable fallbacks),
// validates it, and constructs the Grouper API client that resources and data
// sources will use. Credentials are never stored in state or logged.
func (p *grouperProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config grouperProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reject values that are still unknown at apply time (e.g. sourced from
	// another resource's computed output). Provider config must be known.
	if config.BaseURL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("base_url"),
			"Unknown Grouper base URL",
			"The base_url value is unknown at configuration time. Set it to a known value or via "+envBaseURL+".")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve each value: explicit config wins, otherwise fall back to env.
	baseURL := firstNonEmpty(config.BaseURL, envBaseURL)
	username := firstNonEmpty(config.Username, envUser)
	password := firstNonEmpty(config.Password, envPass)
	timeoutStr := firstNonEmpty(config.RequestTimeout, envTimeout)

	if baseURL == "" {
		resp.Diagnostics.AddAttributeError(path.Root("base_url"),
			"Missing Grouper base URL",
			"Set the base_url provider attribute or the "+envBaseURL+" environment variable.")
	}

	// Parse the timeout, defaulting to 30s.
	timeout := 30 * time.Second
	if timeoutStr != "" {
		parsed, err := time.ParseDuration(timeoutStr)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("request_timeout"),
				"Invalid request_timeout",
				"request_timeout must be a Go duration string like \"30s\" or \"1m\". Error: "+err.Error())
		} else if parsed <= 0 {
			resp.Diagnostics.AddAttributeError(path.Root("request_timeout"),
				"Invalid request_timeout",
				"request_timeout must be greater than zero.")
		} else {
			timeout = parsed
		}
	}

	insecure := config.InsecureSkipVerify.ValueBool()
	if insecure {
		resp.Diagnostics.AddAttributeWarning(path.Root("insecure_skip_verify"),
			"TLS certificate verification disabled",
			"insecure_skip_verify is true. TLS certificates will NOT be verified, exposing connections "+
				"to man-in-the-middle attacks. Only use this against trusted development endpoints.")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Choose an authenticator. Basic auth is the most common Grouper Web
	// Services configuration; the client's Authenticator interface allows
	// swapping this out per deployment without touching resource code.
	var auth client.Authenticator = client.NoAuth{}
	if username != "" {
		auth = client.BasicAuth{Username: username, Password: password}
	}

	c, err := client.New(client.Config{
		BaseURL:            baseURL,
		WSVersion:          firstNonEmpty(config.WSVersion, ""),
		Auth:               auth,
		Timeout:            timeout,
		InsecureSkipVerify: insecure,
		UserAgent:          "terraform-provider-grouper/" + p.version,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to configure Grouper client",
			"Could not construct the Grouper API client: "+err.Error())
		return
	}

	tflog.Debug(ctx, "configured grouper provider", map[string]any{
		"base_url":             c.BaseURL(),
		"auth_scheme":          auth.Scheme(),
		"request_timeout":      timeout.String(),
		"insecure_skip_verify": insecure,
	})

	// Make the client available to resources and data sources.
	resp.ResourceData = c
	resp.DataSourceData = c
}

// Resources returns the resource types implemented by this provider. Resource
// types are added in later phases (stem, group, membership) once their API
// operations are confirmed.
func (p *grouperProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewStemResource,
		NewGroupResource,
		NewMembershipResource,
	}
}

// DataSources returns the data source types implemented by this provider. Data
// source types are added in a later phase (stem, group, subject).
func (p *grouperProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewStemDataSource,
		NewGroupDataSource,
		NewSubjectDataSource,
	}
}

// firstNonEmpty returns the configured string value if set, otherwise the value
// of the named environment variable (trimmed).
func firstNonEmpty(configVal types.String, envKey string) string {
	if !configVal.IsNull() && !configVal.IsUnknown() {
		if v := strings.TrimSpace(configVal.ValueString()); v != "" {
			return v
		}
	}
	return strings.TrimSpace(os.Getenv(envKey))
}
