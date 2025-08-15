package provider

import (
	"context"
	"fmt"

	"unithost-terraform/internal/newvm"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &operatingSystemsDataSource{}
	_ datasource.DataSourceWithConfigure = &operatingSystemsDataSource{}
)

// NewOperatingSystemsDataSource is a helper function to simplify the provider implementation.
func NewOperatingSystemsDataSource() datasource.DataSource {
	return &operatingSystemsDataSource{}
}

// operatingSystemsDataSource is the data source implementation.
type operatingSystemsDataSource struct {
	client *newvm.Client
}

// operatingSystemsDataSourceModel maps the data source schema data.
type operatingSystemsDataSourceModel struct {
	Tag              types.String           `tfsdk:"tag"`
	Platform         types.String           `tfsdk:"platform"`
	OperatingSystems []operatingSystemModel `tfsdk:"list"`
}

// operatingSystemsModel maps operatingSystems schema data.
type operatingSystemModel struct {
	//ID       types.String `tfsdk:"id"`
	Tag      types.String `tfsdk:"tag"`
	Name     types.String `tfsdk:"name"`
	Platform types.String `tfsdk:"platform"`
}

// Metadata returns the data source type name.
func (d *operatingSystemsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_operating_systems"
}

// Schema defines the schema for the data source.
func (d *operatingSystemsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"tag": schema.StringAttribute{
				Optional: true,
			},
			"platform": schema.StringAttribute{
				Optional: true,
			},
			"list": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						//"id": schema.StringAttribute{
						//	Computed: true,
						//},
						"tag": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"platform": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *operatingSystemsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	state := operatingSystemsDataSourceModel{}

	// Get the config values (especially optional tag input)
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	operatingSystems, err := d.client.GetOperatingSystems()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read NewVM operating systems",
			err.Error(),
		)
		return
	}

	// Map response body to model
	filtered := []operatingSystemModel{}
	for _, os := range operatingSystems {
		if (state.Tag.IsNull() && state.Platform.IsNull()) || // no filtering
			(!state.Tag.IsNull() && os.Tag == state.Tag.ValueString()) || // tag filtering
			(!state.Platform.IsNull() && os.Platform == state.Platform.ValueString()) { // platform filtering
			filtered = append(filtered, operatingSystemModel{
				//ID:       types.StringValue(os.ID),
				Tag:      types.StringValue(os.Tag),
				Name:     types.StringValue(os.Name),
				Platform: types.StringValue(os.Platform),
			})
		}
	}
	state.OperatingSystems = filtered

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *operatingSystemsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*newvm.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *newvm.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}
