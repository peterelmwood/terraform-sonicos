package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/peterelmwood/terraform-sonicos/internal/client"
)

var (
	_ resource.Resource                = (*zoneResource)(nil)
	_ resource.ResourceWithConfigure   = (*zoneResource)(nil)
	_ resource.ResourceWithImportState = (*zoneResource)(nil)
)

// NewZoneResource constructs the sonicos_zone resource.
func NewZoneResource() resource.Resource {
	return &zoneResource{}
}

type zoneResource struct {
	data *providerData
}

type zoneModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	SecurityType   types.String `tfsdk:"security_type"`
	InterfaceTrust types.Bool   `tfsdk:"interface_trust"`
	AutoGenerate   types.Bool   `tfsdk:"auto_generate_access_rules"`
}

func (r *zoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone"
}

func (r *zoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A security zone.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to the zone name.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique zone name. Changing this forces replacement.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"security_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zone security type: `trusted`, `untrusted`, `public`, `encrypted`, or `wireless`.",
				Validators: []validator.String{
					stringvalidator.OneOf("trusted", "untrusted", "public", "encrypted", "wireless"),
				},
			},
			"interface_trust": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow interface-to-interface communication within the zone (auto-add interface trust). Defaults to `false`.",
			},
			"auto_generate_access_rules": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Auto-generate the default access rules for this zone. Defaults to `true`.",
			},
		},
	}
}

func (r *zoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.data = configureProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *zoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan zoneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj := plan.toAPI()
	if err := r.data.client.CreateZone(ctx, obj); err != nil {
		resp.Diagnostics.AddError("Failed to create zone", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fromAPIZone(&obj))...)
}

func (r *zoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state zoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, err := r.data.client.GetZone(ctx, state.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read zone", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fromAPIZone(obj))...)
}

func (r *zoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan zoneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj := plan.toAPI()
	if err := r.data.client.UpdateZone(ctx, obj.Name, obj); err != nil {
		resp.Diagnostics.AddError("Failed to update zone", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fromAPIZone(&obj))...)
}

func (r *zoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state zoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.DeleteZone(ctx, state.Name.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Failed to delete zone", err.Error())
			return
		}
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
}

func (r *zoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (m zoneModel) toAPI() client.Zone {
	return client.Zone{
		Name:           m.Name.ValueString(),
		SecurityType:   m.SecurityType.ValueString(),
		InterfaceTrust: m.InterfaceTrust.ValueBool(),
		AutoGenerate:   m.AutoGenerate.ValueBool(),
	}
}

func fromAPIZone(obj *client.Zone) zoneModel {
	return zoneModel{
		ID:             types.StringValue(obj.Name),
		Name:           types.StringValue(obj.Name),
		SecurityType:   types.StringValue(obj.SecurityType),
		InterfaceTrust: types.BoolValue(obj.InterfaceTrust),
		AutoGenerate:   types.BoolValue(obj.AutoGenerate),
	}
}
