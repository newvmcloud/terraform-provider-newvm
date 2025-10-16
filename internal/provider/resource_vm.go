package provider

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"unithost-terraform/internal/newvm"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &vmResource{}
	_ resource.ResourceWithConfigure   = &vmResource{}
	_ resource.ResourceWithImportState = &vmResource{}
)

// NewVmResource is a helper function to simplify the provider implementation.
func NewVmResource() resource.Resource {
	return &vmResource{}
}

// vmResourceModel maps the resource schema data.
type vmResourceModel struct {
	ID          types.String `tfsdk:"id"`
	VmProductID types.String `tfsdk:"product"`
	Os          types.String `tfsdk:"os"`
	Hostname    types.String `tfsdk:"hostname"`
	Location    types.String `tfsdk:"location"`
	Ram         types.Int64  `tfsdk:"ram"`
	Cores       types.Int64  `tfsdk:"cores"`
	Disk        types.Int64  `tfsdk:"disk"`
	Comments    types.String `tfsdk:"comments"`
	SshKey      types.String `tfsdk:"ssh_key"`
	IsVpcOnly   types.Bool   `tfsdk:"is_vpc_only"`
	UseDhcp     types.Bool   `tfsdk:"use_dhcp"`
	Vpc         types.List   `tfsdk:"vpc"`
	IpAddress   types.String `tfsdk:"ip_address"`
	SubnetMask  types.String `tfsdk:"subnet_mask"`
	Gateway     types.String `tfsdk:"gateway"`
	DnsServer   types.String `tfsdk:"dns_server"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// vmResource is the resource implementation.
type vmResource struct {
	client *newvm.Client
}

// Metadata returns the resource type name.
func (r *vmResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

// Schema defines the schema for the resource.
func (r *vmResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VM.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the VM. (order number)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product": schema.StringAttribute{
				Description: "product ID of the VM. (eg. 'VM-A1' or 'VM-B3')",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					RequiresReplaceIfProductPrefixChanges(),
				},
			},
			"os": schema.StringAttribute{
				Description: "operating system for the VM.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				Description: "hostname for the VM.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"location": schema.StringAttribute{
				Description: "datacenter location for the VM.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ram": schema.Int64Attribute{
				Description: "additional memory for the VM in GB.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"cores": schema.Int64Attribute{
				Description: "additional vCPU cores for the VM.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"disk": schema.Int64Attribute{
				Description: "additional harddisk space for the VM in GB.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					PreventDecreaseInt64{Attr: "disk", Unit: "GB"},
				},
			},
			"comments": schema.StringAttribute{
				Description: "Comments for the VM order.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ssh_key": schema.StringAttribute{
				Description: "SSH key to use for administrator account.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_vpc_only": schema.BoolAttribute{
				Description: "Indicates if VM is only connected to a VPC.",
				Optional:    true,
				// Computed:    true,
				// Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"use_dhcp": schema.BoolAttribute{
				Description: "Indicates if VM should use DHCP for obtaining IP data.",
				Optional:    true,
				// Computed:    true,
				// Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"vpc": schema.ListAttribute{
				Description: "List of VPC numbers (VxLANs) attached to the VM.",
				Optional:    true,
				ElementType: types.Int32Type,
			},
			"ip_address": schema.StringAttribute{
				Description: "IP address of VM's primary network interface.",
				Optional:    true,
				Computed:    true,
				//Default:     stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subnet_mask": schema.StringAttribute{
				Description: "Subnetmask of VM's primary network interface.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gateway": schema.StringAttribute{
				Description: "Default gateway IP address of VM's primary network interface.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dns_server": schema.StringAttribute{
				Description: "DNS server IP address of VM's primary network interface.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the VM.",
				Computed:    true,
			},
		},
	}
}

// Create a new VM resource.
func (r *vmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan vmResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var vpcIDs []int32
	if !plan.Vpc.IsNull() && !plan.Vpc.IsUnknown() {
		diags = plan.Vpc.ElementsAs(ctx, &vpcIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from plan
	newVmOrder := newvm.Vm{
		VmProductID: plan.VmProductID.ValueString(),
		Os:          plan.Os.ValueString(),
		Hostname:    plan.Hostname.ValueString(),
		Location:    plan.Location.ValueString(),
		Ram:         plan.Ram.ValueInt64(),
		Cores:       int(plan.Cores.ValueInt64()),
		HdSize:      plan.Disk.ValueInt64(),
		SshKey:      plan.SshKey.ValueString(),
		IsVpcOnly:   plan.IsVpcOnly.ValueBool(),
		UseDhcp:     plan.UseDhcp.ValueBool(),
		Vpc:         vpcIDs,
		IpAddress:   plan.IpAddress.ValueString(),
		SubnetMask:  plan.SubnetMask.ValueString(),
		Gateway:     plan.Gateway.ValueString(),
		DnsServer:   plan.DnsServer.ValueString(),
	}

	// Create new vm
	vm, err := r.client.CreateVm(newVmOrder)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating VM",
			"Could not create VM, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(strconv.Itoa(vm.OrderID))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.

func (r *vmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state vmResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmId := state.ID.ValueString()
	if vmId != "" {
		log.Println("Reading VM: ", vmId)

		// Get refreshed vm value from NewVM
		vm, err := r.client.GetVm(vmId)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading VM",
				"Could not read VM "+vmId+": "+err.Error(),
			)
			return
		}
		// vm.Vpc is []int32 coming from API
		list, diags := types.ListValueFrom(ctx, types.Int32Type, vm.Vpc)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Overwrite items with refreshed state
		state.VmProductID = types.StringValue(vm.VmProductID)
		state.Os = types.StringValue(vm.Os)
		state.Location = types.StringValue(vm.Location)
		state.Hostname = types.StringValue(vm.Hostname)
		state.Ram = types.Int64Value(vm.Ram)
		state.Cores = types.Int64Value(int64(vm.Cores))
		state.Disk = types.Int64Value(vm.HdSize)
		state.Vpc = list
		state.IpAddress = types.StringValue(vm.IpAddress)
		state.Gateway = types.StringValue(vm.Gateway)
		state.DnsServer = types.StringValue(vm.DnsServer)
		state.SubnetMask = types.StringValue(vm.SubnetMask)

		// Set refreshed state
		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			log.Printf("Error updating state: %v", resp.Diagnostics.Errors())
			return
		}
	}
}

func (r *vmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan vmResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch current VM data from API
	vmCurrent, err := r.client.GetVm(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading NewVM VM",
			"Could not read NewVM VM ID "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	var vpcIDs []int32
	if !plan.Vpc.IsNull() && !plan.Vpc.IsUnknown() {
		diags = plan.Vpc.ElementsAs(ctx, &vpcIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from plan
	vmUpdated := newvm.Vm{
		VmProductID: plan.VmProductID.ValueString(),
		Os:          plan.Os.ValueString(),
		Hostname:    plan.Hostname.ValueString(),
		Location:    plan.Location.ValueString(),
		Ram:         plan.Ram.ValueInt64(),
		Cores:       int(plan.Cores.ValueInt64()),
		HdSize:      plan.Disk.ValueInt64(),
		SshKey:      plan.SshKey.ValueString(),
		IsVpcOnly:   plan.IsVpcOnly.ValueBool(),
		UseDhcp:     plan.UseDhcp.ValueBool(),
		Vpc:         vpcIDs,
	}

	// Static IPs only when applicable, and only if changed
	if plan.IsVpcOnly.ValueBool() && !plan.UseDhcp.ValueBool() {
		if !plan.IpAddress.IsUnknown() && !plan.IpAddress.IsNull() {
			vmUpdated.IpAddress = plan.IpAddress.ValueString()
		}
		if !plan.SubnetMask.IsUnknown() && !plan.SubnetMask.IsNull() {
			vmUpdated.SubnetMask = plan.SubnetMask.ValueString()
		}
		if !plan.Gateway.IsUnknown() && !plan.Gateway.IsNull() {
			vmUpdated.Gateway = plan.Gateway.ValueString()
		}
		if !plan.DnsServer.IsUnknown() && !plan.DnsServer.IsNull() {
			vmUpdated.DnsServer = plan.DnsServer.ValueString()
		}
	}

	// Update existing VM
	_, errUpdate := r.client.UpdateVm(plan.ID.ValueString(), vmCurrent, vmUpdated)
	if errUpdate != nil {
		resp.Diagnostics.AddError(
			"Error Updating NewVM Vm",
			"Could not update VM, unexpected error: "+errUpdate.Error(),
		)
		return
	}

	// Fetch updated items from GetVm as UpdateVm items are not populated.
	vmNew, errGet := r.client.GetVm(plan.ID.ValueString())
	if errGet != nil {
		resp.Diagnostics.AddError(
			"Error Reading NewVM VM",
			"Could not read NewVM VM ID "+plan.ID.ValueString()+": "+errGet.Error(),
		)
		return
	}

	// vm.Vpc is []int32 coming from API
	list, diags := types.ListValueFrom(ctx, types.Int32Type, vmNew.Vpc)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update resource state with updated items and timestamp
	plan.VmProductID = types.StringValue(vmNew.VmProductID)
	plan.Os = types.StringValue(vmNew.Os)
	plan.Location = types.StringValue(vmNew.Location)
	plan.Hostname = types.StringValue(vmNew.Hostname)
	plan.Ram = types.Int64Value(vmNew.Ram)
	plan.Cores = types.Int64Value(int64(vmNew.Cores))
	plan.Disk = types.Int64Value(vmNew.HdSize)
	plan.SshKey = types.StringValue(vmNew.SshKey)
	plan.Vpc = list
	plan.IpAddress = types.StringValue(vmNew.IpAddress)
	plan.Gateway = types.StringValue(vmNew.Gateway)
	plan.DnsServer = types.StringValue(vmNew.DnsServer)
	plan.SubnetMask = types.StringValue(vmNew.SubnetMask)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *vmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state vmResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	vmID := state.ID.ValueString()
	if vmID != "" {
		// Delete existing vm
		err := r.client.DeleteVm(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Deleting VM",
				"Could not delete VM, unexpected error: "+err.Error(),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Error Deleting VM",
			"Could not delete VM, no ID given",
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *vmResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type productPrefixReplaceModifier struct{}

func (m productPrefixReplaceModifier) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	// Skip if unknown or null
	if req.PlanValue.IsUnknown() || req.StateValue.IsUnknown() ||
		req.PlanValue.IsNull() || req.StateValue.IsNull() {
		return
	}

	planStr := req.PlanValue.ValueString()
	stateStr := req.StateValue.ValueString()

	if len(planStr) >= 4 && len(stateStr) >= 4 {
		prefixPlan := planStr[:4]
		prefixState := stateStr[:4]

		if (prefixPlan == "VM-A" && prefixState == "VM-B") ||
			(prefixPlan == "VM-B" && prefixState == "VM-A") {
			resp.RequiresReplace = true
		}
	}
}

func (m productPrefixReplaceModifier) Description(_ context.Context) string {
	return "Requires replacement if product code prefix changes between 'VM-A' and 'VM-B'."
}

func (m productPrefixReplaceModifier) MarkdownDescription(_ context.Context) string {
	return m.Description(context.Background())
}

// Exported function to use in schema
func RequiresReplaceIfProductPrefixChanges() planmodifier.String {
	return productPrefixReplaceModifier{}
}
