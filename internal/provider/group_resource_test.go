package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func groupSchema(t *testing.T) fwresource.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewGroupResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("group schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestGroupResource_Schema(t *testing.T) {
	t.Parallel()
	s := groupSchema(t)

	for _, name := range []string{"id", "name", "display_name", "description", "stem_name"} {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing group attribute %q", name)
			continue
		}
		if attr.GetDescription() == "" {
			t.Errorf("attribute %q has no description", name)
		}
	}

	if !s.Attributes["id"].IsComputed() {
		t.Error("id must be computed")
	}
	if !s.Attributes["name"].IsRequired() {
		t.Error("name must be required")
	}
}

func TestGroupResource_ImplementsImport(t *testing.T) {
	t.Parallel()
	if _, ok := NewGroupResource().(resource.ResourceWithImportState); !ok {
		t.Error("group resource must implement ResourceWithImportState")
	}
}
