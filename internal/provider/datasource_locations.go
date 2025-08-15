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
	_ datasource.DataSource              = &locationsDataSource{}
	_ datasource.DataSourceWithConfigure = &locationsDataSource{}
)

// NewLocationsDataSource is a helper function to simplify the provider implementation.
func NewLocationsDataSource() datasource.DataSource {
	return &locationsDataSource{}
}

// locationsDataSource is the data source implementation.
type locationsDataSource struct {
	client *newvm.Client
}

// locationsDataSourceModel maps the data source schema data.
type locationsDataSourceModel struct {
	Code      types.String    `tfsdk:"code"`
	Locations []locationModel `tfsdk:"list"`
}

// locationsModel maps locations schema data.
type locationModel struct {
	//ID         types.String   `tfsdk:"id"`
	Name       types.String   `tfsdk:"name"`
	Code       types.String   `tfsdk:"code"`
	ProductIds []types.String `tfsdk:"product_ids"`
}

// Metadata returns the data source type name.
func (d *locationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_locations"
}

// Schema defines the schema for the data source.
func (d *locationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"code": schema.StringAttribute{
				Optional: true,
			},
			"list": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						//"id": schema.StringAttribute{
						//	Computed: true,
						//},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"code": schema.StringAttribute{
							Computed: true,
						},
						"product_ids": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *locationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	state := locationsDataSourceModel{}

	// Get the config values (especially optional code input)
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	locations, err := d.client.GetLocations()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read NewVM locations",
			err.Error(),
		)
		return
	}

	// Map response body to model
	filtered := []locationModel{}
	for _, location := range locations {
		if (state.Code.IsNull()) || // no filtering
			(!state.Code.IsNull() && location.Code == state.Code.ValueString()) { // code filtering
			var productIds []types.String
			for _, pid := range location.ProductIds {
				productIds = append(productIds, types.StringValue(pid))
			}
			filtered = append(filtered, locationModel{
				//ID:         types.StringValue(location.ID),
				Name:       types.StringValue(location.Name),
				Code:       types.StringValue(location.Code),
				ProductIds: productIds,
			})
		}
	}
	state.Locations = filtered

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *locationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
