package provider

import (
	"context"

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
	_ resource.Resource                = (*serviceObjectResource)(nil)
	_ resource.ResourceWithConfigure   = (*serviceObjectResource)(nil)
	_ resource.ResourceWithImportState = (*serviceObjectResource)(nil)
)

// NewServiceObjectResource constructs the sonicos_service_object resource.
func NewServiceObjectResource() resource.Resource {
	return &serviceObjectResource{}
}

type serviceObjectResource struct {
	data *providerData
}

type serviceObjectModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Protocol  types.String `tfsdk:"protocol"`
	PortBegin types.Int64  `tfsdk:"port_begin"`
	PortEnd   types.Int64  `tfsdk:"port_end"`
}

func (r *serviceObjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_object"
}

func (r *serviceObjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A custom service object. For `tcp` and `udp` services, specify a port or port range; `icmp` and other protocols ignore the port attributes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to the service object name.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique service object name. Changing this forces replacement.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"protocol": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "IP protocol: `tcp`, `udp`, or `icmp`.",
				Validators: []validator.String{
					stringvalidator.OneOf("tcp", "udp", "icmp"),
				},
			},
			"port_begin": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "First port in the range (1-65535). Required for `tcp`/`udp`.",
			},
			"port_end": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Last port in the range (1-65535). Defaults to `port_begin` for a single port.",
			},
		},
	}
}

func (r *serviceObjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.data = configureProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *serviceObjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceObjectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, diags := plan.toAPI()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.CreateServiceObject(ctx, obj); err != nil {
		resp.Diagnostics.AddError("Failed to create service object", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fromAPIServiceObject(&obj))...)
}

func (r *serviceObjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceObjectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, err := r.data.client.GetServiceObject(ctx, state.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read service object", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fromAPIServiceObject(obj))...)
}

func (r *serviceObjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceObjectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, diags := plan.toAPI()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.UpdateServiceObject(ctx, obj.Name, obj); err != nil {
		resp.Diagnostics.AddError("Failed to update service object", err.Error())
		return
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fromAPIServiceObject(&obj))...)
}

func (r *serviceObjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceObjectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.data.client.DeleteServiceObject(ctx, state.Name.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Failed to delete service object", err.Error())
			return
		}
	}
	commitIfEnabled(ctx, r.data, &resp.Diagnostics)
}

func (r *serviceObjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (m serviceObjectModel) toAPI() (client.ServiceObject, diag.Diagnostics) {
	var diags diag.Diagnostics
	obj := client.ServiceObject{
		Name:     m.Name.ValueString(),
		Protocol: m.Protocol.ValueString(),
	}

	if m.Protocol.ValueString() == "tcp" || m.Protocol.ValueString() == "udp" {
		if m.PortBegin.IsNull() {
			diags.AddError("Missing port", "`port_begin` is required for `tcp` and `udp` service objects.")
			return obj, diags
		}
		begin := m.PortBegin.ValueInt64()
		end := begin
		if !m.PortEnd.IsNull() {
			end = m.PortEnd.ValueInt64()
		}
		obj.Port = &client.ServicePort{Begin: begin, End: end}
	}
	return obj, diags
}

func fromAPIServiceObject(obj *client.ServiceObject) serviceObjectModel {
	m := serviceObjectModel{
		ID:        types.StringValue(obj.Name),
		Name:      types.StringValue(obj.Name),
		Protocol:  types.StringValue(obj.Protocol),
		PortBegin: types.Int64Null(),
		PortEnd:   types.Int64Null(),
	}
	if obj.Port != nil {
		m.PortBegin = types.Int64Value(obj.Port.Begin)
		m.PortEnd = types.Int64Value(obj.Port.End)
	}
	return m
}
