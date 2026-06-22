package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/peterelmwood/terraform-sonicos/internal/client"
)

var (
	_ resource.Resource                = (*natPolicyResource)(nil)
	_ resource.ResourceWithConfigure   = (*natPolicyResource)(nil)
	_ resource.ResourceWithImportState = (*natPolicyResource)(nil)
)

// NewNATPolicyResource constructs the sonicos_nat_policy resource.
func NewNATPolicyResource() resource.Resource {
	return &natPolicyResource{}
}

type natPolicyResource struct {
	data *providerData
}

// natPolicyModel is the Terraform state model for an IPv4 NAT policy. Each
// object slot is a name; an empty original_* / inbound / outbound means "any",
// and an empty translated_* means "original" (no translation on that field).
type natPolicyModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Enable                types.Bool   `tfsdk:"enable"`
	Comment               types.String `tfsdk:"comment"`
	OriginalSource        types.String `tfsdk:"original_source"`
	TranslatedSource      types.String `tfsdk:"translated_source"`
	OriginalDestination   types.String `tfsdk:"original_destination"`
	TranslatedDestination types.String `tfsdk:"translated_destination"`
	OriginalService       types.String `tfsdk:"original_service"`
	TranslatedService     types.String `tfsdk:"translated_service"`
	InboundInterface      types.String `tfsdk:"inbound_interface"`
	OutboundInterface     types.String `tfsdk:"outbound_interface"`
}

func (r *natPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nat_policy"
}

func (r *natPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	originalDesc := "address/service object name, or empty for `any`."
	translatedDesc := "address/service object name, or empty for `original` (no translation)."
	resp.Schema = schema.Schema{
		MarkdownDescription: "An IPv4 NAT policy. The appliance assigns a UUID on creation, used as the resource `id`. The six object slots reference address and service objects by name; original slots default to `any` and translated slots default to `original` (no translation) when left empty.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Appliance-assigned NAT policy UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Policy name. Used to locate the policy's UUID immediately after creation, so it must be unique.",
			},
			"enable": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the policy is enabled. Defaults to `true`.",
			},
			"comment": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Free-form comment.",
			},
			"original_source":        natStringAttr("Original source " + originalDesc),
			"translated_source":      natStringAttr("Translated source " + translatedDesc),
			"original_destination":   natStringAttr("Original destination " + originalDesc),
			"translated_destination": natStringAttr("Translated destination " + translatedDesc),
			"original_service":       natStringAttr("Original service " + originalDesc),
			"translated_service":     natStringAttr("Translated service " + translatedDesc),
			"inbound_interface":      natStringAttr("Inbound interface name, or empty for `any`."),
			"outbound_interface":     natStringAttr("Outbound interface name, or empty for `any`."),
		},
	}
}

// natStringAttr builds the optional/computed string attribute shared by all NAT
// object slots, each defaulting to empty (the appliance's wildcard).
func natStringAttr(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: desc,
	}
}

func (r *natPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.data = configureProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *natPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan natPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid, err := r.data.client.CreateNATPolicy(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create NAT policy", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(uuid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *natPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state natPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.data.client.GetNATPolicy(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read NAT policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fromAPINATPolicy(policy))...)
}

func (r *natPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan natPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := plan.ID.ValueString()
	policy := plan.toAPI()
	policy.UUID = uuid
	if err := r.data.client.UpdateNATPolicy(ctx, uuid, policy); err != nil {
		resp.Diagnostics.AddError("Failed to update NAT policy", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *natPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state natPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.DeleteNATPolicy(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Failed to delete NAT policy", err.Error())
			return
		}
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
}

func (r *natPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// natOriginalRef builds a reference for an original (untranslated) slot:
// a named object, or "any" when empty.
func natOriginalRef(name string) *client.NATRef {
	if name == "" {
		return &client.NATRef{Any: true}
	}
	return &client.NATRef{Name: name}
}

// natTranslatedRef builds a reference for a translated slot: a named object, or
// "original" (no translation) when empty.
func natTranslatedRef(name string) *client.NATRef {
	if name == "" {
		return &client.NATRef{Original: true}
	}
	return &client.NATRef{Name: name}
}

func (m natPolicyModel) toAPI() client.NATPolicy {
	return client.NATPolicy{
		Name:                  m.Name.ValueString(),
		Enable:                m.Enable.ValueBool(),
		Comment:               m.Comment.ValueString(),
		OriginalSource:        natOriginalRef(m.OriginalSource.ValueString()),
		TranslatedSource:      natTranslatedRef(m.TranslatedSource.ValueString()),
		OriginalDestination:   natOriginalRef(m.OriginalDestination.ValueString()),
		TranslatedDestination: natTranslatedRef(m.TranslatedDestination.ValueString()),
		OriginalService:       natOriginalRef(m.OriginalService.ValueString()),
		TranslatedService:     natTranslatedRef(m.TranslatedService.ValueString()),
		Inbound:               natOriginalRef(m.InboundInterface.ValueString()),
		Outbound:              natOriginalRef(m.OutboundInterface.ValueString()),
	}
}

// natRefName returns the object name for a reference, or "" for the "any" /
// "original" built-ins.
func natRefName(ref *client.NATRef) string {
	if ref == nil || ref.Any || ref.Original {
		return ""
	}
	return ref.Name
}

func fromAPINATPolicy(policy *client.NATPolicy) natPolicyModel {
	return natPolicyModel{
		ID:                    types.StringValue(policy.UUID),
		Name:                  types.StringValue(policy.Name),
		Enable:                types.BoolValue(policy.Enable),
		Comment:               types.StringValue(policy.Comment),
		OriginalSource:        types.StringValue(natRefName(policy.OriginalSource)),
		TranslatedSource:      types.StringValue(natRefName(policy.TranslatedSource)),
		OriginalDestination:   types.StringValue(natRefName(policy.OriginalDestination)),
		TranslatedDestination: types.StringValue(natRefName(policy.TranslatedDestination)),
		OriginalService:       types.StringValue(natRefName(policy.OriginalService)),
		TranslatedService:     types.StringValue(natRefName(policy.TranslatedService)),
		InboundInterface:      types.StringValue(natRefName(policy.Inbound)),
		OutboundInterface:     types.StringValue(natRefName(policy.Outbound)),
	}
}
