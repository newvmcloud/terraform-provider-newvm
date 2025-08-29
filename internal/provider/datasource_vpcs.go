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
	_ datasource.DataSource              = &vpcsDataSource{}
	_ datasource.DataSourceWithConfigure = &vpcsDataSource{}
)

// NewVpcsDataSource is a helper function to simplify the provider implementation.
func NewVpcsDataSource() datasource.DataSource {
	return &vpcsDataSource{}
}

// vpcsDataSource is the data source implementation.
type vpcsDataSource struct {
	client *newvm.Client
}

// vpcsDataSourceModel maps the data source schema data.
type vpcsDataSourceModel struct {
	Number types.Int32 `tfsdk:"number"`
	Vpcs   []vpcModel  `tfsdk:"list"`
}

// vpcsModel maps vpcs schema data.
type vpcModel struct {
	//ID         types.String   `tfsdk:"id"`
	Name    types.String     `tfsdk:"name"`
	Number  types.Int32      `tfsdk:"number"`
	Members []vpcMemberModel `tfsdk:"members"`
}

// vpcMember
type vpcMemberModel struct {
	//ID         types.String   `tfsdk:"id"`
	OrderId    types.Int32  `tfsdk:"orderid"`
	MacAddress types.String `tfsdk:"macaddress"`
}

// Metadata returns the data source type name.
func (d *vpcsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpcs"
}

// Schema defines the schema for the data source.
func (d *vpcsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"number": schema.Int32Attribute{
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
						"number": schema.Int32Attribute{
							Computed: true,
						},
						"members": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"orderid": schema.Int32Attribute{
										Computed: true,
									},
									"macaddress": schema.StringAttribute{
										Computed: true,
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
func (d *vpcsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	state := vpcsDataSourceModel{}

	// Get the config values (especially optional code input)
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpcs, err1 := d.client.GetVpcs()
	if err1 != nil {
		resp.Diagnostics.AddError(
			"Unable to read NewVM VPCs",
			err1.Error(),
		)
		return
	}

	vpcMembers, err2 := d.client.GetVpcMembers()
	if err2 != nil {
		resp.Diagnostics.AddError(
			"Unable to read NewVM VPC members",
			err2.Error(),
		)
		return
	}

	// Map response body to model
	filtered := []vpcModel{}
	for _, vpc := range vpcs {
		if (state.Number.IsNull()) || // no filtering
			(!state.Number.IsNull() && vpc.Number == state.Number.ValueInt32()) { // number filtering
			var members []vpcMemberModel
			for _, member := range vpcMembers {
				if vpc.Number == member.Vxlan {
					vpcMember := vpcMemberModel{
						OrderId:    types.Int32Value(int32(member.OrderId)),
						MacAddress: types.StringValue(member.MacAddress),
					}
					members = append(members, vpcMember)
				}
			}
			filtered = append(filtered, vpcModel{
				//ID:         types.StringValue(vpc.ID),
				Name:    types.StringValue(vpc.Name),
				Number:  types.Int32Value(vpc.Number),
				Members: members,
			})
		}
	}
	state.Vpcs = filtered

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *vpcsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
