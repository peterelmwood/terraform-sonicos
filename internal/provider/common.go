package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// known reports whether a string attribute has a concrete, usable value — i.e.
// it is neither null nor unknown. Validators use it to skip attributes that are
// absent or not yet resolved (e.g. derived from another resource during plan).
func known(s types.String) bool {
	return !s.IsNull() && !s.IsUnknown()
}

// configureProviderData extracts the shared providerData from a resource or
// data source ConfigureRequest. providerData is nil during early framework
// lifecycle calls, which is not an error. It reports a clear diagnostic if the
// framework ever passes an unexpected type.
func configureProviderData(raw any, diags *diag.Diagnostics) *providerData {
	if raw == nil {
		return nil
	}
	data, ok := raw.(*providerData)
	if !ok {
		diags.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *providerData, got %T. This is a bug in the provider.", raw),
		)
		return nil
	}
	return data
}

// commitIfEnabled activates the SonicOS pending configuration when the provider
// is configured to commit on apply. Resources call this after a successful
// write so the change goes live within the same apply.
func commitIfEnabled(ctx context.Context, data *providerData, diags *diag.Diagnostics) {
	if data == nil || !data.commitOnApply {
		return
	}
	if err := data.client.CommitPending(ctx); err != nil {
		diags.AddError(
			"Failed to commit SonicOS pending configuration",
			"The change was staged but committing the pending configuration failed. The appliance may have uncommitted changes.\n\n"+err.Error(),
		)
	}
}
