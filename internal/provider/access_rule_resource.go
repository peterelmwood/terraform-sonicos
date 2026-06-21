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
	_ resource.Resource                = (*accessRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*accessRuleResource)(nil)
	_ resource.ResourceWithImportState = (*accessRuleResource)(nil)
)

// NewAccessRuleResource constructs the sonicos_access_rule resource.
func NewAccessRuleResource() resource.Resource {
	return &accessRuleResource{}
}

type accessRuleResource struct {
	data *providerData
}

// accessRuleModel is the Terraform state model for an IPv4 access rule. The
// source/destination address and service default to "any" when their name
// attribute is empty, which is the common firewall-rule case.
type accessRuleModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	From        types.String `tfsdk:"from"`
	To          types.String `tfsdk:"to"`
	Action      types.String `tfsdk:"action"`
	Enable      types.Bool   `tfsdk:"enable"`
	SourceName  types.String `tfsdk:"source_name"`
	DestName    types.String `tfsdk:"destination_name"`
	ServiceName types.String `tfsdk:"service_name"`
	Comment     types.String `tfsdk:"comment"`
}

func (r *accessRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_rule"
}

func (r *accessRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An IPv4 access (firewall) rule between two zones. The appliance assigns a UUID on creation, which is used as the resource `id`. Leave `source_name`, `destination_name`, or `service_name` empty to match `any`.",
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

func (r *accessRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.data = configureProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *accessRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid, err := r.data.client.CreateAccessRule(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create access rule", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(uuid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *accessRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.data.client.GetAccessRule(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read access rule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fromAPIAccessRule(rule))...)
}

func (r *accessRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan accessRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := plan.ID.ValueString()
	rule := plan.toAPI()
	rule.UUID = uuid
	if err := r.data.client.UpdateAccessRule(ctx, uuid, rule); err != nil {
		resp.Diagnostics.AddError("Failed to update access rule", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *accessRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.DeleteAccessRule(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Failed to delete access rule", err.Error())
			return
		}
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
}

func (r *accessRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by appliance-assigned UUID; Read populates the rest.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// nameOrAny builds an AccessRuleName, defaulting to "any" when name is empty.
func nameOrAny(name string) *client.AccessRuleName {
	if name == "" {
		return &client.AccessRuleName{Any: true}
	}
	return &client.AccessRuleName{Name: name}
}

func (m accessRuleModel) toAPI() client.AccessRule {
	return client.AccessRule{
		Name:        m.Name.ValueString(),
		From:        m.From.ValueString(),
		To:          m.To.ValueString(),
		Action:      m.Action.ValueString(),
		Enable:      m.Enable.ValueBool(),
		Source:      nameOrAny(m.SourceName.ValueString()),
		Destination: nameOrAny(m.DestName.ValueString()),
		Service:     nameOrAny(m.ServiceName.ValueString()),
		Comment:     m.Comment.ValueString(),
	}
}

// nameValue returns the object name, or "" for an "any" reference.
func nameValue(n *client.AccessRuleName) string {
	if n == nil || n.Any {
		return ""
	}
	return n.Name
}

func fromAPIAccessRule(rule *client.AccessRule) accessRuleModel {
	return accessRuleModel{
		ID:          types.StringValue(rule.UUID),
		Name:        types.StringValue(rule.Name),
		From:        types.StringValue(rule.From),
		To:          types.StringValue(rule.To),
		Action:      types.StringValue(rule.Action),
		Enable:      types.BoolValue(rule.Enable),
		SourceName:  types.StringValue(nameValue(rule.Source)),
		DestName:    types.StringValue(nameValue(rule.Destination)),
		ServiceName: types.StringValue(nameValue(rule.Service)),
		Comment:     types.StringValue(rule.Comment),
	}
}
