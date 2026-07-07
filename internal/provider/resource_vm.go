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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	OrderID     types.Int64  `tfsdk:"order_id"`
	Name        types.String `tfsdk:"name"`
	VmProductID types.String `tfsdk:"product"`
	Os          types.String `tfsdk:"os"`
	Hostname    types.String `tfsdk:"hostname"`
	Location    types.String `tfsdk:"location"`
	Ram         types.Int64  `tfsdk:"ram"`
	Cores       types.Int64  `tfsdk:"cores"`
	Disk        types.Int64  `tfsdk:"disk"`
	SshKey      types.String `tfsdk:"ssh_key"`
	IsVpcOnly   types.Bool   `tfsdk:"is_vpc_only"`
	UseDhcp     types.Bool   `tfsdk:"use_dhcp"`
	RegisterDns types.Bool   `tfsdk:"register_dns"`
	Vpc         types.Set    `tfsdk:"vpc"`
	IpAddress   types.String `tfsdk:"ip_address"`
	SubnetMask  types.String `tfsdk:"subnet_mask"`
	Gateway     types.String `tfsdk:"gateway"`
	DnsServer   types.String `tfsdk:"dns_server"`
	Metadata    types.Map    `tfsdk:"metadata"`
	// LastUpdated types.String `tfsdk:"last_updated"`
}

// vmResource is the resource implementation.
type vmResource struct {
	client *newvm.Client
}

type productPrefixReplaceModifier struct{}

// the data to wait for after provisioning
type vmProvisioningWait struct {
	UUID bool
	IP   bool
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
				Description: "UUID of the VM",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"order_id": schema.Int64Attribute{
				Description: "order number",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "name of the VM. (eg. 'VM00123')",
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
			"ssh_key": schema.StringAttribute{
				Description: "SSH key to use for administrator account during initial provisioning only.",
				Optional:    true,
			},
			"is_vpc_only": schema.BoolAttribute{
				Description: "Indicates if VM is only connected to a VPC.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"use_dhcp": schema.BoolAttribute{
				Description: "Indicates if VM should use DHCP for obtaining IP data.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"register_dns": schema.BoolAttribute{
				Description: "Indicates if hostname should be registered in DNS as A/AAAA.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"vpc": schema.SetAttribute{
				Description: "List of VPC numbers (VxLANs) attached to the VM.",
				Optional:    true,
				ElementType: types.Int32Type,
			},
			"ip_address": schema.StringAttribute{
				Description: "IP address of VM's primary network interface.",
				Optional:    true,
				Computed:    true,
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
			"metadata": schema.MapAttribute{
				Optional:    true,
				Description: "Order metadata as a map of key => list of values.",
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
			},
			// "last_updated": schema.StringAttribute{
			// Description: "Timestamp of the last Terraform update of the VM.",
			// Computed:    true,
			// PlanModifiers: []planmodifier.String{
			// stringplanmodifier.UseStateForUnknown(),
			// },
			// },
		},
	}
}

// Create a new VM resource.
func (r *vmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
	}

	var plan vmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config vmResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var vpcIDs []int32
	if !plan.Vpc.IsNull() && !plan.Vpc.IsUnknown() {
		resp.Diagnostics.Append(plan.Vpc.ElementsAs(ctx, &vpcIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	newVmOrder := newvm.Vm{
		VmProductID: plan.VmProductID.ValueString(),
		Os:          plan.Os.ValueString(),
		Hostname:    plan.Hostname.ValueString(),
		Location:    plan.Location.ValueString(),
		Ram:         plan.Ram.ValueInt64(),
		Cores:       int(plan.Cores.ValueInt64()),
		HdSize:      plan.Disk.ValueInt64(),
		SshKey:      config.SshKey.ValueString(),
		IsVpcOnly:   plan.IsVpcOnly.ValueBool(),
		UseDhcp:     plan.UseDhcp.ValueBool(),
		RegisterDns: plan.RegisterDns.ValueBool(),
		Vpc:         vpcIDs,
		IpAddress:   plan.IpAddress.ValueString(),
		SubnetMask:  plan.SubnetMask.ValueString(),
		Gateway:     plan.Gateway.ValueString(),
		DnsServer:   plan.DnsServer.ValueString(),
	}

	vm, err := r.client.CreateVm(ctx, newVmOrder)
	if err != nil {
		resp.Diagnostics.AddError("Error creating VM", "Could not create VM, unexpected error: "+err.Error())
		return
	}

	plan.OrderID = types.Int64Value(int64(vm.OrderID))

	metadataItems, d := expandOrderMetadata(ctx, plan.Metadata, int(vm.OrderID))
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SyncOrderMetaData(ctx, int64(vm.OrderID), metadataItems); err != nil {
		resp.Diagnostics.AddError(
			"Error syncing VM metadata",
			"VM was created, but metadata could not be synced: "+err.Error(),
		)
		return
	}

	userSetStatic := plan.IsVpcOnly.ValueBool() && !plan.UseDhcp.ValueBool() &&
		!plan.IpAddress.IsNull() && !plan.IpAddress.IsUnknown() &&
		plan.IpAddress.ValueString() != ""

	provisionedVm, err := r.WaitForProvisioning(ctx, plan.OrderID.ValueInt64(), vmProvisioningWait{
		UUID: true,
		IP:   !userSetStatic,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for VM provisioning",
			"VM order was created, but provisioning data was not ready in time: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(provisionedVm.ID)
	plan.Name = types.StringValue(provisionedVm.VmName)

	if !userSetStatic {
		plan.IpAddress = types.StringValue(provisionedVm.IpAddress)
		plan.Gateway = types.StringValue(provisionedVm.Gateway)
		plan.DnsServer = types.StringValue(provisionedVm.DnsServer)
		plan.SubnetMask = types.StringValue(provisionedVm.SubnetMask)
	}

	// read back the meta data
	metaItems, err := r.client.GetOrderMetaData(ctx, int64(vm.OrderID))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading VM metadata",
			"VM was created, but metadata could not be read back: "+err.Error(),
		)
		return
	}

	plan.Metadata, d = flattenOrderMetadata(ctx, metaItems)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *vmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmOrderID := state.OrderID.ValueInt64()
	if !(vmOrderID > 0) {
		resp.Diagnostics.AddError(
			"Error Reading VM",
			"Could not read VM: no order ID present in state.",
		)
		return
	}

	vm, err := r.client.GetVm(ctx, vmOrderID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VM",
			"Could not read VM "+strconv.Itoa(int(vmOrderID))+": "+err.Error(),
		)
		return
	}

	vpcNumbers := vm.Vpc
	if vpcNumbers == nil {
		vpcNumbers = []int32{}
	}

	set, diags := types.SetValueFrom(ctx, types.Int32Type, vpcNumbers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("VM: %v", vm)

	state.ID = types.StringValue(vm.ID)
	state.OrderID = types.Int64Value(int64(vm.OrderID))
	state.Name = types.StringValue(vm.VmName)
	state.VmProductID = types.StringValue(vm.VmProductID)
	state.Os = types.StringValue(vm.Os)
	if vm.Location != "" {
		state.Location = types.StringValue(vm.Location)
	}
	// state.Hostname = types.StringValue(vm.Hostname)
	state.Ram = types.Int64Value(vm.Ram)
	state.Cores = types.Int64Value(int64(vm.Cores))
	state.Disk = types.Int64Value(vm.HdSize)
	state.Vpc = set
	state.IpAddress = types.StringValue(vm.IpAddress)
	state.Gateway = types.StringValue(vm.Gateway)
	state.DnsServer = types.StringValue(vm.DnsServer)
	state.SubnetMask = types.StringValue(vm.SubnetMask)
	state.IsVpcOnly = types.BoolValue(vm.IsVpcOnly)
	state.UseDhcp = types.BoolValue(vm.UseDhcp)
	// state.RegisterDns = types.BoolValue(vm.RegisterDns)

	// state.SshKey = types.StringNull()

	metaItems, err := r.client.GetOrderMetaData(ctx, vmOrderID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VM metadata",
			"Could not read metadata for VM "+strconv.FormatInt(vmOrderID, 10)+": "+err.Error(),
		)
		return
	}

	state.Metadata, diags = flattenOrderMetadata(ctx, metaItems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error updating state: %v", resp.Diagnostics.Errors())
		return
	}
}

func (r *vmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vmResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmCurrent, err := r.client.GetVm(ctx, plan.OrderID.ValueInt64())
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

	vmUpdated := newvm.Vm{
		VmProductID: plan.VmProductID.ValueString(),
		Os:          plan.Os.ValueString(),
		// Hostname:    plan.Hostname.ValueString(),
		Location:  plan.Location.ValueString(),
		Ram:       plan.Ram.ValueInt64(),
		Cores:     int(plan.Cores.ValueInt64()),
		HdSize:    plan.Disk.ValueInt64(),
		IsVpcOnly: plan.IsVpcOnly.ValueBool(),
		UseDhcp:   plan.UseDhcp.ValueBool(),
		// RegisterDns: plan.RegisterDns.ValueBool(),
		Vpc: vpcIDs,
	}

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

	_, errUpdate := r.client.UpdateVm(ctx, plan.OrderID.ValueInt64(), vmCurrent, vmUpdated)
	if errUpdate != nil {
		resp.Diagnostics.AddError(
			"Error Updating NewVM Vm",
			"Could not update VM, unexpected error: "+errUpdate.Error(),
		)
		return
	}

	metadataItems, d := expandOrderMetadata(ctx, plan.Metadata, int(plan.OrderID.ValueInt64()))
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SyncOrderMetaData(ctx, plan.OrderID.ValueInt64(), metadataItems); err != nil {
		resp.Diagnostics.AddError(
			"Error syncing VM metadata",
			"Could not sync metadata for VM: "+err.Error(),
		)
		return
	}

	vmNew, errGet := r.client.GetVm(ctx, plan.OrderID.ValueInt64())
	if errGet != nil {
		resp.Diagnostics.AddError(
			"Error Reading NewVM VM",
			"Could not read NewVM VM ID "+plan.ID.ValueString()+": "+errGet.Error(),
		)
		return
	}

	metaItems, err := r.client.GetOrderMetaData(ctx, plan.OrderID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading VM metadata",
			"Could not read metadata after update: "+err.Error(),
		)
		return
	}

	plan.Metadata, d = flattenOrderMetadata(ctx, metaItems)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpcNumbers := vmNew.Vpc
	if vpcNumbers == nil {
		vpcNumbers = []int32{}
	}

	set, diags := types.SetValueFrom(ctx, types.Int32Type, vpcNumbers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.VmProductID = types.StringValue(vmNew.VmProductID)
	plan.Os = types.StringValue(vmNew.Os)
	if vmNew.Location != "" {
		plan.Location = types.StringValue(vmNew.Location)
	}
	// plan.Hostname = types.StringValue(vmNew.Hostname)
	plan.Ram = types.Int64Value(vmNew.Ram)
	plan.Cores = types.Int64Value(int64(vmNew.Cores))
	plan.Disk = types.Int64Value(vmNew.HdSize)
	plan.Vpc = set
	plan.IpAddress = types.StringValue(vmNew.IpAddress)
	plan.Gateway = types.StringValue(vmNew.Gateway)
	plan.DnsServer = types.StringValue(vmNew.DnsServer)
	plan.SubnetMask = types.StringValue(vmNew.SubnetMask)
	plan.IsVpcOnly = types.BoolValue(vmNew.IsVpcOnly)
	plan.UseDhcp = types.BoolValue(vmNew.UseDhcp)
	// plan.RegisterDns = types.BoolValue(vmNew.RegisterDns)
	// plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// plan.SshKey = types.StringNull()

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *vmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmOrderID := state.OrderID.ValueInt64()
	if !(vmOrderID > 0) {
		resp.Diagnostics.AddError(
			"Error Deleting VM",
			"Could not delete VM, no ID given",
		)
		return
	}

	err := r.client.DeleteVm(ctx, vmOrderID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting VM",
			"Could not delete VM, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *vmResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	orderID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected numeric order ID, got %q: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("order_id"), orderID)...)
}

func (m productPrefixReplaceModifier) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
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

func (r *vmResource) WaitForProvisioning(ctx context.Context, orderID int64, wait vmProvisioningWait) (*newvm.Vm, error) {
	backoff := 300 * time.Millisecond
	maxBackoff := 5 * time.Second

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		vm, err := r.client.GetVm(ctx, orderID)
		if err == nil && vm != nil {
			ready := true

			if wait.UUID && vm.ID == "" {
				ready = false
			}
			if wait.IP && vm.IpAddress == "" {
				ready = false
			}

			if ready {
				return vm, nil
			}
		}

		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (m productPrefixReplaceModifier) Description(_ context.Context) string {
	return "Requires replacement if product code prefix changes between 'VM-A' and 'VM-B'."
}

func (m productPrefixReplaceModifier) MarkdownDescription(_ context.Context) string {
	return m.Description(context.Background())
}

// RequiresReplaceIfProductPrefixChanges returns the string plan modifier.
func RequiresReplaceIfProductPrefixChanges() planmodifier.String {
	return productPrefixReplaceModifier{}
}
