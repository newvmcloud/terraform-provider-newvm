package provider

import (
	"context"
	"os"
	"unithost-terraform/internal/newvm"

	// "unithost-terraform/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &newvmProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &newvmProvider{
			Version: version,
		}
	}
}

// newvmProviderModel maps provider schema data to a Go type.
type newvmProviderModel struct {
	Host     types.String `tfsdk:"host"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Totp     types.String `tfsdk:"totp"`
}

// newvmProvider is the provider implementation.
type newvmProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	Version string
}

// Metadata returns the provider type name.
func (p *newvmProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "newvm"
	resp.Version = p.Version
}

// Schema defines the provider-level schema for configuration data.
func (p *newvmProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with NewVM.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "URI for NewVM API. May also be provided via NEWVM_HOST environment variable.",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "Username for NewVM API. May also be provided via NEWVM_USERNAME environment variable.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Password for NewVM API. May also be provided via NEWVM_PASSWORD environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"totp": schema.StringAttribute{
				Description: "TOTP token for NewVM API.",
				Optional:    true,
			},
		},
	}
}

// Configure prepares a NewVM API client for data sources and resources.
func (p *newvmProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring NewVM client")

	// Retrieve provider data from configuration
	var config newvmProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown NewVM API Host",
			"The provider cannot create the NewVM API client as there is an unknown configuration value for the NewVM API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the NEWVM_HOST environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown NewVM API Username",
			"The provider cannot create the NewVM API client as there is an unknown configuration value for the NewVM API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the NEWVM_USERNAME environment variable.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown NewVM API Password",
			"The provider cannot create the NewVM API client as there is an unknown configuration value for the NewVM API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the NEWVM_PASSWORD environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	host := os.Getenv("NEWVM_HOST")
	username := os.Getenv("NEWVM_USERNAME")
	password := os.Getenv("NEWVM_PASSWORD")
	totp := ""

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	if !config.Totp.IsNull() {
		totp = config.Totp.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing NewVM API Host",
			"The provider cannot create the NewVM API client as there is a missing or empty value for the NewVM API host. "+
				"Set the host value in the configuration or use the NEWVM_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing NewVM API Username",
			"The provider cannot create the NewVM API client as there is a missing or empty value for the NewVM API username. "+
				"Set the username value in the configuration or use the NEWVM_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing NewVM API Password",
			"The provider cannot create the NewVM API client as there is a missing or empty value for the NewVM API password. "+
				"Set the password value in the configuration or use the NEWVM_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "newvm_host", host)
	ctx = tflog.SetField(ctx, "newvm_username", username)
	ctx = tflog.SetField(ctx, "newvm_password", password)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "newvm_password")

	tflog.Debug(ctx, "Creating NewVM client")

	// Create a new NewVM client using the configuration values
	client, err := newvm.NewClient(&host, &username, &password, &totp)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create NewVM API Client",
			"An unexpected error occurred when creating the NewVM API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"NewVM Client Error: "+err.Error(),
		)
		return
	}

	// Make the NewVM client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured NewVM client with token: "+client.Token, map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *newvmProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewControlPanelProductsDataSource,
		NewLocationsDataSource,
		NewOperatingSystemsDataSource,
		NewVmProductsDataSource,
		NewVpcsDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *newvmProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewControlPanelResource,
		NewVmResource,
		NewVpcResource,
	}
}
