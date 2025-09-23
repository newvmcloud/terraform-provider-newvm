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
	_ datasource.DataSource              = &controlPanelProductsDataSource{}
	_ datasource.DataSourceWithConfigure = &controlPanelProductsDataSource{}
)

// NewControlPanelProductsDataSource is a helper function to simplify the provider implementation.
func NewControlPanelProductsDataSource() datasource.DataSource {
	return &controlPanelProductsDataSource{}
}

// controlPanelProductsDataSource is the data source implementation.
type controlPanelProductsDataSource struct {
	client *newvm.Client
}

// controlPanelProductsDataSourceModel maps the data source schema data.
type controlPanelProductsDataSourceModel struct {
	ID                   types.String               `tfsdk:"id"`
	ControlPanelProducts []controlPanelProductModel `tfsdk:"list"`
}

// controlPanelProductsDataSourceModel maps the data source schema data.
type controlPanelExtensionModel struct {
	ID          types.String  `tfsdk:"id"`
	Description types.String  `tfsdk:"description"`
	Price       types.Float64 `tfsdk:"price"`
}

// controlPanelProductModel maps controlPanelProduct schema data.
type controlPanelProductModel struct {
	ID          types.String                 `tfsdk:"id"`
	Type        types.String                 `tfsdk:"type"`
	Description types.String                 `tfsdk:"description"`
	Price       types.Float64                `tfsdk:"price"`
	Extensions  []controlPanelExtensionModel `tfsdk:"extensions"`
}

// Metadata returns the data source type name.
func (d *controlPanelProductsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_control_panel_products"
}

// Schema defines the schema for the data source.
func (d *controlPanelProductsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
			},
			"list": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"type": schema.StringAttribute{
							Computed: true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
						"price": schema.Float64Attribute{
							Computed:    true,
							Description: "Base price of the control panel.",
						},
						"extensions": schema.ListNestedAttribute{
							Optional:    true,
							Description: "Optional extensions for the control panel.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Required: true,
									},
									"description": schema.StringAttribute{
										Computed: true,
									},
									"price": schema.Float64Attribute{
										Computed:    true,
										Description: "Extra price of the control panel extension.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *controlPanelProductsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	state := controlPanelProductsDataSourceModel{}

	// Get the config values (especially optional tag input)
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	controlPanelProducts, err := d.client.GetControlPanelProducts()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read NewVM Control panel products",
			err.Error(),
		)
		return
	}

	// Map response body to model
	filtered := []controlPanelProductModel{}
	for _, controlPanelProduct := range controlPanelProducts {
		if (state.ID.IsNull()) || // no filtering
			(!state.ID.IsNull() && controlPanelProduct.ID == state.ID.ValueString()) { // ID filtering
			extensions := []controlPanelExtensionModel{}
			for _, extension := range controlPanelProduct.Extensions {
				extensions = append(extensions, controlPanelExtensionModel{
					ID:          types.StringValue(extension.ID),
					Description: types.StringValue(extension.Description),
					Price:       types.Float64Value(extension.Price),
				})
			}
			filtered = append(filtered, controlPanelProductModel{
				ID:          types.StringValue(controlPanelProduct.ID),
				Type:        types.StringValue(controlPanelProduct.Type),
				Description: types.StringValue(controlPanelProduct.Description),
				Price:       types.Float64Value(controlPanelProduct.Price),
				Extensions:  extensions,
			})
		}
	}
	state.ControlPanelProducts = filtered

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *controlPanelProductsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
