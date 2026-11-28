package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestSchemas_documented requires every resource, data source, and attribute to
// carry a Markdown description, since those descriptions are the published
// Registry documentation.
func TestSchemas_documented(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := New("test")()

	for _, newResource := range p.Resources(ctx) {
		r := newResource()
		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "grouper"}, &meta)
		var resp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &resp)

		if resp.Schema.MarkdownDescription == "" {
			t.Errorf("resource %s has no MarkdownDescription", meta.TypeName)
		}
		for name, attr := range resp.Schema.Attributes {
			if attr.GetMarkdownDescription() == "" {
				t.Errorf("resource %s attribute %q has no MarkdownDescription", meta.TypeName, name)
			}
		}
	}

	for _, newDataSource := range p.DataSources(ctx) {
		d := newDataSource()
		var meta datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "grouper"}, &meta)
		var resp datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &resp)

		if resp.Schema.MarkdownDescription == "" {
			t.Errorf("data source %s has no MarkdownDescription", meta.TypeName)
		}
		for name, attr := range resp.Schema.Attributes {
			if attr.GetMarkdownDescription() == "" {
				t.Errorf("data source %s attribute %q has no MarkdownDescription", meta.TypeName, name)
			}
		}
	}
}
