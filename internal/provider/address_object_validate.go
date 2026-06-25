package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/peterelmwood/terraform-sonicos/internal/lint"
)

// The address object resource validates the format of its IP-bearing attributes
// during plan, so mistakes like a malformed host IP or a non-contiguous subnet
// mask surface as a clear Terraform diagnostic instead of an opaque appliance
// rejection at apply time. This mirrors the format rules the standalone linter
// applies across the whole configuration; here they run per-resource and in-plan.

var _ resource.ResourceWithValidateConfig = (*addressObjectResource)(nil)

func (r *addressObjectResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg addressObjectModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch cfg.Type.ValueString() {
	case "host":
		if known(cfg.Host) && !lint.IsIPv4(cfg.Host.ValueString()) {
			resp.Diagnostics.AddAttributeError(path.Root("host"), "Invalid host address",
				fmt.Sprintf("%q is not a valid IPv4 address.", cfg.Host.ValueString()))
		}
	case "network":
		if known(cfg.NetworkSubnet) && !lint.IsIPv4(cfg.NetworkSubnet.ValueString()) {
			resp.Diagnostics.AddAttributeError(path.Root("network_subnet"), "Invalid network address",
				fmt.Sprintf("%q is not a valid IPv4 address.", cfg.NetworkSubnet.ValueString()))
		}
		if known(cfg.NetworkMask) && !lint.IsIPv4Mask(cfg.NetworkMask.ValueString()) {
			resp.Diagnostics.AddAttributeError(path.Root("network_mask"), "Invalid subnet mask",
				fmt.Sprintf("%q is not a valid contiguous IPv4 subnet mask (e.g. 255.255.255.0).", cfg.NetworkMask.ValueString()))
		}
	case "range":
		beginOK := known(cfg.RangeBegin) && lint.IsIPv4(cfg.RangeBegin.ValueString())
		endOK := known(cfg.RangeEnd) && lint.IsIPv4(cfg.RangeEnd.ValueString())
		if known(cfg.RangeBegin) && !lint.IsIPv4(cfg.RangeBegin.ValueString()) {
			resp.Diagnostics.AddAttributeError(path.Root("range_begin"), "Invalid range start",
				fmt.Sprintf("%q is not a valid IPv4 address.", cfg.RangeBegin.ValueString()))
		}
		if known(cfg.RangeEnd) && !lint.IsIPv4(cfg.RangeEnd.ValueString()) {
			resp.Diagnostics.AddAttributeError(path.Root("range_end"), "Invalid range end",
				fmt.Sprintf("%q is not a valid IPv4 address.", cfg.RangeEnd.ValueString()))
		}
		if beginOK && endOK && !lint.AddrLessOrEqual(cfg.RangeBegin.ValueString(), cfg.RangeEnd.ValueString()) {
			resp.Diagnostics.AddAttributeError(path.Root("range_end"), "Invalid range order",
				fmt.Sprintf("range_begin %q must not be greater than range_end %q.", cfg.RangeBegin.ValueString(), cfg.RangeEnd.ValueString()))
		}
	}
}
