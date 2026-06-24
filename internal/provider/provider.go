// Package provider implements the Terraform provider for SonicOS 7+.
//
// It is built on the Terraform Plugin Framework and maps resources one-to-one
// onto SonicOS REST API objects. The provider's Configure establishes the API
// session and hands every resource a shared *client.Client together with the
// commit policy.
//
// The pending/commit model. SonicOS does not apply writes immediately; they
// accumulate in a pending configuration that must be activated with a single
// commit call. Terraform may execute resource operations concurrently (subject
// to the dependency graph and -parallelism) and gives the provider no global
// "end of apply" hook, so the pragmatic policy (controlled by the provider's
// commit_on_apply attribute, default true) is to commit after each resource
// write. That keeps every apply self-consistent at the cost of one commit per
// changed resource. Because SonicOS permits only a single API session, runs
// against one appliance should be serialized regardless. Set commit_on_apply = false to
// stage changes without committing and drive the commit yourself (for example
// with a null_resource calling the API, or a manual commit), which restores the
// "stage everything, commit once" transaction boundary when you need it.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/peterelmwood/terraform-sonicos/internal/client"
)

// Ensure the implementation satisfies the framework interface.
var _ provider.Provider = (*sonicosProvider)(nil)

// providerData is shared with every resource and data source via Configure. It
// bundles the authenticated client with the commit policy so each resource can
// decide whether to activate the pending configuration after a write.
type providerData struct {
	client        *client.Client
	commitOnApply bool
}

type sonicosProvider struct {
	// version is set by the build (via ldflags / goreleaser) and surfaced in
	// the user agent and acceptance-test wiring.
	version string
}

// New returns a function that constructs the provider, the signature
// providerserver expects.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &sonicosProvider{version: version}
	}
}

func (p *sonicosProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sonicos"
	resp.Version = p.version
}

// providerModel mirrors the provider block schema.
type providerModel struct {
	Host          types.String `tfsdk:"host"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	Insecure      types.Bool   `tfsdk:"insecure"`
	CommitOnApply types.Bool   `tfsdk:"commit_on_apply"`
}

func (p *sonicosProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage SonicOS 7+ firewall configuration through the appliance's built-in REST API.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Appliance management URL or host, e.g. `https://192.0.2.1:8443`. May also be set with the `SONICOS_HOST` environment variable.",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Full-admin API username. May also be set with the `SONICOS_USERNAME` environment variable. SonicOS allows only one API session at a time, so use a dedicated automation account.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "API password. May also be set with the `SONICOS_PASSWORD` environment variable.",
			},
			"insecure": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Skip TLS certificate verification. Useful for appliances using the default self-signed certificate. May also be set with the `SONICOS_INSECURE` environment variable. Defaults to `false`.",
			},
			"commit_on_apply": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Activate the SonicOS pending configuration after every resource write. Defaults to `true`. Set to `false` to stage changes without committing and drive the commit yourself.",
			},
		},
	}
}

func (p *sonicosProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Unknown attribute values (e.g. derived from another resource that has not
	// been applied yet) cannot be resolved at configure time. Fail fast rather
	// than silently falling back to environment variables, which would make the
	// provider's effective configuration non-deterministic.
	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("host"), "Unknown host", "The `host` value is unknown at configure time. Set it to a known value (or via SONICOS_HOST) rather than a value derived from another resource.")
	}
	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("username"), "Unknown username", "The `username` value is unknown at configure time. Set it to a known value (or via SONICOS_USERNAME).")
	}
	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("password"), "Unknown password", "The `password` value is unknown at configure time. Set it to a known value (or via SONICOS_PASSWORD).")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration may be supplied directly or via environment variables, with
	// the configuration value taking precedence when both are set.
	host := firstNonEmpty(config.Host, os.Getenv("SONICOS_HOST"))
	username := firstNonEmpty(config.Username, os.Getenv("SONICOS_USERNAME"))
	password := firstNonEmpty(config.Password, os.Getenv("SONICOS_PASSWORD"))

	insecure := false
	if !config.Insecure.IsNull() {
		insecure = config.Insecure.ValueBool()
	} else if os.Getenv("SONICOS_INSECURE") == "true" {
		insecure = true
	}

	commitOnApply := true
	if !config.CommitOnApply.IsNull() {
		commitOnApply = config.CommitOnApply.ValueBool()
	}

	if host == "" {
		resp.Diagnostics.AddAttributeError(path.Root("host"), "Missing host", "Set the provider `host` attribute or the SONICOS_HOST environment variable.")
	}
	if username == "" {
		resp.Diagnostics.AddAttributeError(path.Root("username"), "Missing username", "Set the provider `username` attribute or the SONICOS_USERNAME environment variable.")
	}
	if password == "" {
		resp.Diagnostics.AddAttributeError(path.Root("password"), "Missing password", "Set the provider `password` attribute or the SONICOS_PASSWORD environment variable.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := client.New(client.Config{
		Host:     host,
		Username: username,
		Password: password,
		Insecure: insecure,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create SonicOS client", err.Error())
		return
	}

	if err := c.Login(ctx); err != nil {
		resp.Diagnostics.AddError(
			"Unable to authenticate to SonicOS",
			"Login failed. Verify host, credentials, and that no other API session is active (SonicOS permits only one).\n\n"+err.Error(),
		)
		return
	}

	data := &providerData{client: c, commitOnApply: commitOnApply}
	resp.ResourceData = data
	resp.DataSourceData = data
}

func (p *sonicosProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAddressObjectResource,
		NewAddressObjectIPv6Resource,
		NewServiceObjectResource,
		NewZoneResource,
		NewAccessRuleResource,
		NewAccessRuleIPv6Resource,
		NewNATPolicyResource,
		NewInterfaceResource,
	}
}

func (p *sonicosProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAddressObjectDataSource,
	}
}

func firstNonEmpty(v types.String, fallback string) string {
	if !v.IsNull() && v.ValueString() != "" {
		return v.ValueString()
	}
	return fallback
}
