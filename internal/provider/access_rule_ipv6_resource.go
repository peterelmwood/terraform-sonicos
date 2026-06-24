package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/peterelmwood/terraform-sonicos/internal/client"
)

var (
	_ resource.Resource                = (*accessRuleV6Resource)(nil)
	_ resource.ResourceWithConfigure   = (*accessRuleV6Resource)(nil)
	_ resource.ResourceWithImportState = (*accessRuleV6Resource)(nil)
)

// NewAccessRuleIPv6Resource constructs the sonicos_access_rule_ipv6 resource.
func NewAccessRuleIPv6Resource() resource.Resource {
	return &accessRuleV6Resource{}
}

type accessRuleV6Resource struct {
	data *providerData
}

func (r *accessRuleV6Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_rule_ipv6"
}

func (r *accessRuleV6Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An IPv6 access (firewall) rule between two zones. Identical in shape to `sonicos_access_rule` but operates on the IPv6 rule base. Leave `source_name`, `destination_name`, or `service_name` empty to match `any`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Appliance-assigned rule UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Rule name. Used to locate the rule's UUID immediately after creation, so it must be unique.",
			},
			"from": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source zone, e.g. `LAN`.",
			},
			"to": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Destination zone, e.g. `WAN`.",
			},
			"action": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Rule action: `allow`, `deny`, or `discard`.",
				Validators: []validator.String{
					stringvalidator.OneOf("allow", "deny", "discard"),
				},
			},
			"enable": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the rule is enabled. Defaults to `true`.",
			},
			"source_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Source address object name. Empty means `any`.",
			},
			"destination_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Destination address object name. Empty means `any`.",
			},
			"service_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Service object name. Empty means `any`.",
			},
			"comment": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Free-form comment.",
			},
		},
	}
}

func (r *accessRuleV6Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.data = configureProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *accessRuleV6Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid, err := r.data.client.CreateAccessRuleIPv6(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create IPv6 access rule", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(uuid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *accessRuleV6Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.data.client.GetAccessRuleIPv6(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read IPv6 access rule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fromAPIAccessRule(rule))...)
}

func (r *accessRuleV6Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan accessRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := plan.ID.ValueString()
	rule := plan.toAPI()
	rule.UUID = uuid
	if err := r.data.client.UpdateAccessRuleIPv6(ctx, uuid, rule); err != nil {
		resp.Diagnostics.AddError("Failed to update IPv6 access rule", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *accessRuleV6Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.DeleteAccessRuleIPv6(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Failed to delete IPv6 access rule", err.Error())
			return
		}
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
}

func (r *accessRuleV6Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
