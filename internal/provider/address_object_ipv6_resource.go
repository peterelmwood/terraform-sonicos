package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/peterelmwood/terraform-sonicos/internal/client"
)

var (
	_ resource.Resource                = (*addressObjectV6Resource)(nil)
	_ resource.ResourceWithConfigure   = (*addressObjectV6Resource)(nil)
	_ resource.ResourceWithImportState = (*addressObjectV6Resource)(nil)
)

// NewAddressObjectIPv6Resource constructs the sonicos_address_object_ipv6 resource.
func NewAddressObjectIPv6Resource() resource.Resource {
	return &addressObjectV6Resource{}
}

type addressObjectV6Resource struct {
	data *providerData
}

// addressObjectV6Model mirrors addressObjectModel but uses an IPv6 prefix length
// for networks instead of a dotted mask, and does not offer the FQDN type.
type addressObjectV6Model struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Zone          types.String `tfsdk:"zone"`
	Type          types.String `tfsdk:"type"`
	Host          types.String `tfsdk:"host"`
	NetworkSubnet types.String `tfsdk:"network_subnet"`
	NetworkPrefix types.Int64  `tfsdk:"network_prefix"`
	RangeBegin    types.String `tfsdk:"range_begin"`
	RangeEnd      types.String `tfsdk:"range_end"`
}

func (r *addressObjectV6Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_address_object_ipv6"
}

func (r *addressObjectV6Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An IPv6 address object. The `type` selects which value attributes apply: `host` uses `host`; `network` uses `network_subnet` and `network_prefix`; `range` uses `range_begin` and `range_end`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to the address object name.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique address object name. Changing this forces replacement.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"zone": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Security zone the object is assigned to, e.g. `LAN`, `WAN`, `DMZ`.",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Address object type: `host`, `network`, or `range`. Changing this forces replacement.",
				Validators: []validator.String{
					stringvalidator.OneOf("host", "network", "range"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"host": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Host IPv6 address. Required when `type` is `host`.",
			},
			"network_subnet": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "IPv6 network address. Required when `type` is `network`.",
			},
			"network_prefix": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "IPv6 prefix length (0-128). Required when `type` is `network`.",
				Validators: []validator.Int64{
					int64validator.Between(0, 128),
				},
			},
			"range_begin": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "First IPv6 address in the range. Required when `type` is `range`.",
			},
			"range_end": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Last IPv6 address in the range. Required when `type` is `range`.",
			},
		},
	}
}

func (r *addressObjectV6Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.data = configureProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *addressObjectV6Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan addressObjectV6Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, diags := plan.toAPI()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.CreateAddressObjectIPv6(ctx, obj); err != nil {
		resp.Diagnostics.AddError("Failed to create IPv6 address object", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(obj.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *addressObjectV6Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state addressObjectV6Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, err := r.data.client.GetAddressObjectIPv6(ctx, state.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read IPv6 address object", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fromAPIAddressObjectV6(obj))...)
}

func (r *addressObjectV6Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan addressObjectV6Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, diags := plan.toAPI()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.UpdateAddressObjectIPv6(ctx, obj.Name, obj); err != nil {
		resp.Diagnostics.AddError("Failed to update IPv6 address object", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(obj.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *addressObjectV6Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state addressObjectV6Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.DeleteAddressObjectIPv6(ctx, state.Name.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Failed to delete IPv6 address object", err.Error())
			return
		}
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
}

func (r *addressObjectV6Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (m addressObjectV6Model) toAPI() (client.AddressObjectIPv6, diag.Diagnostics) {
	var diags diag.Diagnostics
	obj := client.AddressObjectIPv6{
		Name: m.Name.ValueString(),
		Zone: m.Zone.ValueString(),
	}

	switch m.Type.ValueString() {
	case "host":
		if isBlank(m.Host) {
			diags.AddError("Missing host", "`host` is required when `type` is `host`.")
			return obj, diags
		}
		obj.Host = &client.AddressObjectHost{IP: m.Host.ValueString()}
	case "network":
		if isBlank(m.NetworkSubnet) || m.NetworkPrefix.IsNull() || m.NetworkPrefix.IsUnknown() {
			diags.AddError("Missing network attributes", "`network_subnet` and `network_prefix` are required when `type` is `network`.")
			return obj, diags
		}
		obj.Network = &client.AddressObjectV6Network{Subnet: m.NetworkSubnet.ValueString(), Prefix: m.NetworkPrefix.ValueInt64()}
	case "range":
		if isBlank(m.RangeBegin) || isBlank(m.RangeEnd) {
			diags.AddError("Missing range attributes", "`range_begin` and `range_end` are required when `type` is `range`.")
			return obj, diags
		}
		obj.Range = &client.AddressObjectRange{Begin: m.RangeBegin.ValueString(), End: m.RangeEnd.ValueString()}
	default:
		diags.AddError("Invalid type", fmt.Sprintf("unknown address object type %q", m.Type.ValueString()))
	}
	return obj, diags
}

func fromAPIAddressObjectV6(obj *client.AddressObjectIPv6) addressObjectV6Model {
	m := addressObjectV6Model{
		ID:            types.StringValue(obj.Name),
		Name:          types.StringValue(obj.Name),
		Zone:          types.StringValue(obj.Zone),
		Host:          types.StringNull(),
		NetworkSubnet: types.StringNull(),
		NetworkPrefix: types.Int64Null(),
		RangeBegin:    types.StringNull(),
		RangeEnd:      types.StringNull(),
	}
	switch {
	case obj.Host != nil:
		m.Type = types.StringValue("host")
		m.Host = types.StringValue(obj.Host.IP)
	case obj.Network != nil:
		m.Type = types.StringValue("network")
		m.NetworkSubnet = types.StringValue(obj.Network.Subnet)
		m.NetworkPrefix = types.Int64Value(obj.Network.Prefix)
	case obj.Range != nil:
		m.Type = types.StringValue("range")
		m.RangeBegin = types.StringValue(obj.Range.Begin)
		m.RangeEnd = types.StringValue(obj.Range.End)
	}
	return m
}
