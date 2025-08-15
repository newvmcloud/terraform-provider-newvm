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
	_ datasource.DataSource              = &vmProductsDataSource{}
	_ datasource.DataSourceWithConfigure = &vmProductsDataSource{}
)

// NewVmProductsDataSource is a helper function to simplify the provider implementation.
func NewVmProductsDataSource() datasource.DataSource {
	return &vmProductsDataSource{}
}

// vmProductsDataSource is the data source implementation.
type vmProductsDataSource struct {
	client *newvm.Client
}

// vmProductsDataSourceModel maps the data source schema data.
type vmProductsDataSourceModel struct {
	ID         types.String     `tfsdk:"id"`
	VmProducts []vmProductModel `tfsdk:"list"`
}

// vmProductModel maps vmProduct schema data.
type vmProductModel struct {
	ID        types.String  `tfsdk:"id"`
	ProductID types.String  `tfsdk:"product"`
	Ram       types.Int64   `tfsdk:"ram"`
	Cores     types.Int32   `tfsdk:"cores"`
	HdSize    types.Int64   `tfsdk:"hdsize"`
	Price     types.Float64 `tfsdk:"price"`
}

// Metadata returns the data source type name.
func (d *vmProductsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_products"
}

// Schema defines the schema for the data source.
func (d *vmProductsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
						"product": schema.StringAttribute{
							Computed: true,
						},
						"ram": schema.Int64Attribute{
							Computed: true,
						},
						"cores": schema.Int32Attribute{
							Computed: true,
						},
						"hdsize": schema.Int64Attribute{
							Computed: true,
						},
						"price": schema.Float64Attribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *vmProductsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	state := vmProductsDataSourceModel{}

	// Get the config values (especially optional tag input)
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmProducts, err := d.client.GetVmProducts()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read NewVM VM products",
			err.Error(),
		)
		return
	}

	// Map response body to model
	filtered := []vmProductModel{}
	for _, vmProduct := range vmProducts {
		if (state.ID.IsNull()) || // no filtering
			(!state.ID.IsNull() && vmProduct.ID == state.ID.ValueString()) { // ID filtering
			filtered = append(filtered, vmProductModel{
				ID:        types.StringValue(vmProduct.ID),
				ProductID: types.StringValue(vmProduct.ProductID),
				Ram:       types.Int64Value(vmProduct.Ram),
				Cores:     types.Int32Value(vmProduct.Cores),
				HdSize:    types.Int64Value(vmProduct.HdSize),
				Price:     types.Float64Value(vmProduct.Price),
			})
		}
	}
	state.VmProducts = filtered

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *vmProductsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
