package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestProviderSchema confirms the provider schema builds without diagnostics.
func TestProviderSchema(t *testing.T) {
	p := New("test")()
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("provider schema diagnostics: %v", resp.Diagnostics)
	}
}

// TestResourceSchemas walks every registered resource and confirms its schema
// builds cleanly. This catches malformed attribute definitions at test time.
func TestResourceSchemas(t *testing.T) {
	p := New("test")()
	factories := p.Resources(context.Background())
	if len(factories) == 0 {
		t.Fatal("no resources registered")
	}
	for _, factory := range factories {
		r := factory()
		resp := &resource.SchemaResponse{}
		r.Schema(context.Background(), resource.SchemaRequest{}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("resource schema diagnostics: %v", resp.Diagnostics)
		}
		if len(resp.Schema.Attributes) == 0 {
			t.Errorf("%T: empty schema", r)
		}
		if _, ok := resp.Schema.Attributes["id"]; !ok {
			t.Errorf("%T: missing computed id attribute", r)
		}
	}
}

// TestDataSourceSchemas does the same for data sources.
func TestDataSourceSchemas(t *testing.T) {
	p := New("test")()
	factories := p.DataSources(context.Background())
	for _, factory := range factories {
		d := factory()
		resp := &datasource.SchemaResponse{}
		d.Schema(context.Background(), datasource.SchemaRequest{}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("data source schema diagnostics: %v", resp.Diagnostics)
		}
	}
}

// New must return something satisfying provider.Provider.
func TestNewReturnsProvider(t *testing.T) {
	var _ provider.Provider = New("test")()
}
