package provider

import (
	"context"
	"fmt"
	"log"
	"time"

	"unithost-terraform/internal/newvm"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &vpcResource{}
	_ resource.ResourceWithConfigure   = &vpcResource{}
	_ resource.ResourceWithImportState = &vpcResource{}
)

// NewVpcResource is a helper function to simplify the provider implementation.
func NewVpcResource() resource.Resource {
	return &vpcResource{}
}

// vpcResourceModel maps the resource schema data.
type vpcResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Number      types.Int32  `tfsdk:"number"`
	Name        types.String `tfsdk:"name"`
	OwnerID     types.Int32  `tfsdk:"owner_id"`
	Removable   types.Int32  `tfsdk:"removable"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// vpcResource is the resource implementation.
type vpcResource struct {
	client *newvm.Client
}

// Metadata returns the resource type name.
func (r *vpcResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc"
}

// Schema defines the schema for the resource.
func (r *vpcResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VPC.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the VPC (UUID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"number": schema.Int32Attribute{
				Description: "Number of the VPC (VxLAN).",
				Computed:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the VPC. (eg. 'Internal Network 1')",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"owner_id": schema.Int32Attribute{
				Description: "Account ID of the VPC owner.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"removable": schema.Int32Attribute{
				Description: "Indicates if the VPC owner can remove the VPC.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the VPC.",
				Computed:    true,
			},
		},
	}
}

// Create a new VPC resource.
func (r *vpcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan vpcResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	newVpcOrder := newvm.Vpc{
		Name: plan.Name.ValueString(),
	}

	// Create new VPC
	vpc, err := r.client.CreateVpc(newVpcOrder)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating VPC",
			"Could not create VPC, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(vpc.ID)
	plan.Name = types.StringValue(newVpcOrder.Name)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Try to read back
	if cp, err := r.client.GetVpc(vpc.ID); err == nil {
		plan.Number = types.Int32Value(cp.Number)
		plan.OwnerID = types.Int32Value(cp.OwnerID)
		plan.Removable = types.Int32Value(int32(cp.Removable))
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.

func (r *vpcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state vpcResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpcId := state.ID.ValueString()
	if vpcId != "" {
		log.Println("Reading VPC: ", vpcId)

		// Get refreshed VPC value from NewVM
		vpc, err := r.client.GetVpc(vpcId)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading VPC",
				"Could not read VPC "+vpcId+": "+err.Error(),
			)
			return
		}

		// Overwrite items with refreshed state
		state.Name = types.StringValue(vpc.Name)
		state.Number = types.Int32Value(vpc.Number)
		state.OwnerID = types.Int32Value(vpc.OwnerID)
		state.Removable = types.Int32Value(int32(vpc.Removable))

		// Set refreshed state
		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			log.Printf("Error updating state: %v", resp.Diagnostics.Errors())
			return
		}
	} else {
		return
	}
}

func (r *vpcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan vpcResourceModel
	var prior vpcResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &prior)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	updateVpcOrder := newvm.Vpc{
		Name: plan.Name.ValueString(),
	}

	// Update existing VPC
	err := r.client.UpdateVpc(plan.ID.ValueString(), updateVpcOrder)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NewVM VPC",
			"Could not update VPC, unexpected error: "+err.Error(),
		)
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *vpcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state vpcResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	vpcID := state.ID.ValueString()
	if vpcID != "" {
		// Delete existing VPC
		err := r.client.DeleteVpc(vpcID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Deleting VPC",
				"Could not delete VPC, unexpected error: "+err.Error(),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Error Deleting VPC",
			"Could not delete VPC, no ID given",
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *vpcResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client
}

func (r *vpcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
