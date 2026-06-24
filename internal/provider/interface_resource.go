package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/peterelmwood/terraform-sonicos/internal/client"
)

var (
	_ resource.Resource                = (*interfaceResource)(nil)
	_ resource.ResourceWithConfigure   = (*interfaceResource)(nil)
	_ resource.ResourceWithImportState = (*interfaceResource)(nil)
)

// NewInterfaceResource constructs the sonicos_interface resource.
func NewInterfaceResource() resource.Resource {
	return &interfaceResource{}
}

type interfaceResource struct {
	data *providerData
}

type interfaceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Zone         types.String `tfsdk:"zone"`
	IPAssignment types.String `tfsdk:"ip_assignment"`
	IP           types.String `tfsdk:"ip"`
	Netmask      types.String `tfsdk:"netmask"`
	Gateway      types.String `tfsdk:"gateway"`
	MTU          types.Int64  `tfsdk:"mtu"`
	Comment      types.String `tfsdk:"comment"`
	MgmtHTTPS    types.Bool   `tfsdk:"management_https"`
	MgmtPing     types.Bool   `tfsdk:"management_ping"`
	MgmtSSH      types.Bool   `tfsdk:"management_ssh"`
	MgmtSNMP     types.Bool   `tfsdk:"management_snmp"`
}

func (r *interfaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_interface"
}

func (r *interfaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Configuration of a physical or virtual IPv4 interface (e.g. `X2`). Interfaces are not created or destroyed by this provider: applying configures the named, already-present interface, and destroying simply stops Terraform from managing it (the physical interface and its last-applied settings remain on the appliance).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to the interface name.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Interface name, e.g. `X0`, `X2`. Changing this forces replacement.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"zone": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Security zone the interface is assigned to.",
			},
			"ip_assignment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "IP assignment mode: `static` or `dhcp`.",
				Validators: []validator.String{
					stringvalidator.OneOf("static", "dhcp"),
				},
			},
			"ip": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Static IPv4 address. Required when `ip_assignment` is `static`.",
			},
			"netmask": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Static subnet mask, e.g. `255.255.255.0`. Required when `ip_assignment` is `static`.",
			},
			"gateway": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Default gateway for the interface. Optional, used with `static`.",
			},
			"mtu": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1500),
				MarkdownDescription: "Interface MTU. Defaults to `1500`.",
				Validators: []validator.Int64{
					int64validator.Between(68, 9216),
				},
			},
			"comment": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Free-form comment.",
			},
			"management_https": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow HTTPS management on this interface. Defaults to `false`.",
			},
			"management_ping": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow ICMP ping on this interface. Defaults to `false`.",
			},
			"management_ssh": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow SSH management on this interface. Defaults to `false`.",
			},
			"management_snmp": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow SNMP on this interface. Defaults to `false`.",
			},
		},
	}
}

func (r *interfaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.data = configureProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *interfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan interfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	iface, diags := plan.toAPI()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.ConfigureInterface(ctx, iface); err != nil {
		resp.Diagnostics.AddError("Failed to configure interface", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(iface.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interfaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state interfaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	iface, err := r.data.client.GetInterface(ctx, state.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read interface", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fromAPIInterface(iface))...)
}

func (r *interfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan interfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	iface, diags := plan.toAPI()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.ConfigureInterface(ctx, iface); err != nil {
		resp.Diagnostics.AddError("Failed to configure interface", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(iface.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interfaceResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Interfaces are physical and cannot be deleted via the API. Removing the
	// resource simply stops Terraform from managing it; the framework drops it
	// from state once Delete returns without diagnostics. The interface keeps
	// its last-applied configuration on the appliance.
}

func (r *interfaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (m interfaceModel) toAPI() (client.Interface, diag.Diagnostics) {
	var diags diag.Diagnostics
	iface := client.Interface{
		Name:    m.Name.ValueString(),
		Zone:    m.Zone.ValueString(),
		Comment: m.Comment.ValueString(),
		MTU:     m.MTU.ValueInt64(),
		Management: &client.InterfaceManagement{
			HTTPS: m.MgmtHTTPS.ValueBool(),
			Ping:  m.MgmtPing.ValueBool(),
			SSH:   m.MgmtSSH.ValueBool(),
			SNMP:  m.MgmtSNMP.ValueBool(),
		},
	}

	switch m.IPAssignment.ValueString() {
	case "static":
		if isBlank(m.IP) || isBlank(m.Netmask) {
			diags.AddError("Missing static addressing", "`ip` and `netmask` are required when `ip_assignment` is `static`.")
			return iface, diags
		}
		iface.Static = &client.InterfaceStatic{
			IP:      m.IP.ValueString(),
			Netmask: m.Netmask.ValueString(),
			Gateway: m.Gateway.ValueString(),
		}
	case "dhcp":
		iface.DHCP = true
	}
	return iface, diags
}

func fromAPIInterface(iface *client.Interface) interfaceModel {
	m := interfaceModel{
		ID:        types.StringValue(iface.Name),
		Name:      types.StringValue(iface.Name),
		Zone:      types.StringValue(iface.Zone),
		Comment:   types.StringValue(iface.Comment),
		MTU:       types.Int64Value(iface.MTU),
		IP:        types.StringNull(),
		Netmask:   types.StringNull(),
		Gateway:   types.StringValue(""),
		MgmtHTTPS: types.BoolValue(false),
		MgmtPing:  types.BoolValue(false),
		MgmtSSH:   types.BoolValue(false),
		MgmtSNMP:  types.BoolValue(false),
	}
	if iface.Static != nil {
		m.IPAssignment = types.StringValue("static")
		m.IP = types.StringValue(iface.Static.IP)
		m.Netmask = types.StringValue(iface.Static.Netmask)
		m.Gateway = types.StringValue(iface.Static.Gateway)
	} else {
		m.IPAssignment = types.StringValue("dhcp")
	}
	if iface.Management != nil {
		m.MgmtHTTPS = types.BoolValue(iface.Management.HTTPS)
		m.MgmtPing = types.BoolValue(iface.Management.Ping)
		m.MgmtSSH = types.BoolValue(iface.Management.SSH)
		m.MgmtSNMP = types.BoolValue(iface.Management.SNMP)
	}
	return m
}
