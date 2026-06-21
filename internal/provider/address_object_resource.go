package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource                = (*addressObjectResource)(nil)
	_ resource.ResourceWithConfigure   = (*addressObjectResource)(nil)
	_ resource.ResourceWithImportState = (*addressObjectResource)(nil)
)

// NewAddressObjectResource constructs the sonicos_address_object resource.
func NewAddressObjectResource() resource.Resource {
	return &addressObjectResource{}
}

type addressObjectResource struct {
	data *providerData
}

// addressObjectModel is the Terraform state model for an IPv4 address object.
// The type discriminator is explicit; only the fields relevant to the chosen
// type may be set.
type addressObjectModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Zone          types.String `tfsdk:"zone"`
	Type          types.String `tfsdk:"type"`
	Host          types.String `tfsdk:"host"`
	NetworkSubnet types.String `tfsdk:"network_subnet"`
	NetworkMask   types.String `tfsdk:"network_mask"`
	RangeBegin    types.String `tfsdk:"range_begin"`
	RangeEnd      types.String `tfsdk:"range_end"`
	FQDNDomain    types.String `tfsdk:"fqdn_domain"`
}

func (r *addressObjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_address_object"
}

func (r *addressObjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An IPv4 address object. The `type` selects which value attributes apply: `host` uses `host`; `network` uses `network_subnet` and `network_mask`; `range` uses `range_begin` and `range_end`; `fqdn` uses `fqdn_domain`.",
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
				MarkdownDescription: "Address object type: `host`, `network`, `range`, or `fqdn`. Changing this forces replacement.",
				Validators: []validator.String{
					stringvalidator.OneOf("host", "network", "range", "fqdn"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"host": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Host IP address. Required when `type` is `host`.",
			},
			"network_subnet": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Network address. Required when `type` is `network`.",
			},
			"network_mask": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Network mask, e.g. `255.255.255.0`. Required when `type` is `network`.",
			},
			"range_begin": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "First IP in the range. Required when `type` is `range`.",
			},
			"range_end": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Last IP in the range. Required when `type` is `range`.",
			},
			"fqdn_domain": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Fully qualified domain name. Required when `type` is `fqdn`.",
			},
		},
	}
}

func (r *addressObjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.data = configureProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *addressObjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan addressObjectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, diags := plan.toAPI()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.CreateAddressObjectIPv4(ctx, obj); err != nil {
		resp.Diagnostics.AddError("Failed to create address object", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(obj.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *addressObjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state addressObjectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, err := r.data.client.GetAddressObjectIPv4(ctx, state.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Object removed out-of-band; drop it from state so a plan recreates it.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read address object", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fromAPIAddressObject(obj))...)
}

func (r *addressObjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan addressObjectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, diags := plan.toAPI()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.UpdateAddressObjectIPv4(ctx, obj.Name, obj); err != nil {
		resp.Diagnostics.AddError("Failed to update address object", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(obj.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *addressObjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state addressObjectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.DeleteAddressObjectIPv4(ctx, state.Name.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Failed to delete address object", err.Error())
			return
		}
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
}

func (r *addressObjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by address object name; Read populates the remaining attributes.
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// toAPI converts the Terraform model into the API object, validating that the
// attributes required by the chosen type are present.
func (m addressObjectModel) toAPI() (client.AddressObjectIPv4, diag.Diagnostics) {
	var diags diag.Diagnostics
	obj := client.AddressObjectIPv4{
		Name: m.Name.ValueString(),
		Zone: m.Zone.ValueString(),
	}

	switch m.Type.ValueString() {
	case "host":
		if m.Host.IsNull() || m.Host.ValueString() == "" {
			diags.AddError("Missing host", "`host` is required when `type` is `host`.")
			return obj, diags
		}
		obj.Host = &client.AddressObjectHost{IP: m.Host.ValueString()}
	case "network":
		if m.NetworkSubnet.IsNull() || m.NetworkMask.IsNull() {
			diags.AddError("Missing network attributes", "`network_subnet` and `network_mask` are required when `type` is `network`.")
			return obj, diags
		}
		obj.Network = &client.AddressObjectNetwork{Subnet: m.NetworkSubnet.ValueString(), Mask: m.NetworkMask.ValueString()}
	case "range":
		if m.RangeBegin.IsNull() || m.RangeEnd.IsNull() {
			diags.AddError("Missing range attributes", "`range_begin` and `range_end` are required when `type` is `range`.")
			return obj, diags
		}
		obj.Range = &client.AddressObjectRange{Begin: m.RangeBegin.ValueString(), End: m.RangeEnd.ValueString()}
	case "fqdn":
		if m.FQDNDomain.IsNull() || m.FQDNDomain.ValueString() == "" {
			diags.AddError("Missing fqdn_domain", "`fqdn_domain` is required when `type` is `fqdn`.")
			return obj, diags
		}
		obj.FQDN = &client.AddressObjectFQDN{Domain: m.FQDNDomain.ValueString()}
	default:
		diags.AddError("Invalid type", fmt.Sprintf("unknown address object type %q", m.Type.ValueString()))
	}
	return obj, diags
}

// fromAPIAddressObject maps an API object back into the Terraform model,
// inferring the type discriminator from which value is populated.
func fromAPIAddressObject(obj *client.AddressObjectIPv4) addressObjectModel {
	m := addressObjectModel{
		ID:   types.StringValue(obj.Name),
		Name: types.StringValue(obj.Name),
		Zone: types.StringValue(obj.Zone),
		// Value attributes default to null and are set per type below.
		Host:          types.StringNull(),
		NetworkSubnet: types.StringNull(),
		NetworkMask:   types.StringNull(),
		RangeBegin:    types.StringNull(),
		RangeEnd:      types.StringNull(),
		FQDNDomain:    types.StringNull(),
	}
	switch {
	case obj.Host != nil:
		m.Type = types.StringValue("host")
		m.Host = types.StringValue(obj.Host.IP)
	case obj.Network != nil:
		m.Type = types.StringValue("network")
		m.NetworkSubnet = types.StringValue(obj.Network.Subnet)
		m.NetworkMask = types.StringValue(obj.Network.Mask)
	case obj.Range != nil:
		m.Type = types.StringValue("range")
		m.RangeBegin = types.StringValue(obj.Range.Begin)
		m.RangeEnd = types.StringValue(obj.Range.End)
	case obj.FQDN != nil:
		m.Type = types.StringValue("fqdn")
		m.FQDNDomain = types.StringValue(obj.FQDN.Domain)
	}
	return m
}
