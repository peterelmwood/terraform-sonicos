package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var (
	_ datasource.DataSource              = (*addressObjectDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*addressObjectDataSource)(nil)
)

// NewAddressObjectDataSource constructs the sonicos_address_object data source,
// which looks up an existing IPv4 address object by name.
func NewAddressObjectDataSource() datasource.DataSource {
	return &addressObjectDataSource{}
}

type addressObjectDataSource struct {
	data *providerData
}

func (d *addressObjectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_address_object"
}

func (d *addressObjectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an existing IPv4 address object by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to the address object name.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the address object to look up.",
			},
			"zone": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Security zone the object is assigned to.",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Address object type: `host`, `network`, `range`, or `fqdn`.",
			},
			"host": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Host IP address (when `type` is `host`).",
			},
			"network_subnet": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Network address (when `type` is `network`).",
			},
			"network_mask": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Network mask (when `type` is `network`).",
			},
			"range_begin": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "First IP in the range (when `type` is `range`).",
			},
			"range_end": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last IP in the range (when `type` is `range`).",
			},
			"fqdn_domain": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Fully qualified domain name (when `type` is `fqdn`).",
			},
		},
	}
}

func (d *addressObjectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.data = configureProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *addressObjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config addressObjectModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, err := d.data.client.GetAddressObjectIPv4(ctx, config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read address object", err.Error())
		return
	}

	// Reuse the resource mapping; the model fields line up one-to-one.
	state := fromAPIAddressObject(obj)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
