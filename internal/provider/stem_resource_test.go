package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func stemSchema(t *testing.T) fwresource.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewStemResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("stem schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestStemResource_Schema(t *testing.T) {
	t.Parallel()
	s := stemSchema(t)

	for _, name := range []string{"id", "name", "display_name", "description", "parent_stem_name"} {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing stem attribute %q", name)
			continue
		}
		if attr.GetDescription() == "" && attr.GetMarkdownDescription() == "" {
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

func TestStemResource_ImplementsImport(t *testing.T) {
	t.Parallel()
	if _, ok := NewStemResource().(resource.ResourceWithImportState); !ok {
		t.Error("stem resource must implement ResourceWithImportState")
	}
}

func TestParentStemName(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"example:apps:myteam:azure": "example:apps:myteam",
		"example":                   "",
		"a:b":                       "a",
	}
	for in, want := range cases {
		if got := parentStemName(in); got != want {
			t.Errorf("parentStemName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGrouperPathPattern(t *testing.T) {
	t.Parallel()
	valid := []string{"example", "example:apps", "example:apps:myteam", "a:b:c-d_e.f"}
	invalid := []string{"", ":example", "example:", "example::apps", "example apps", "example:app s"}
	for _, v := range valid {
		if !grouperPathPattern.MatchString(v) {
			t.Errorf("expected %q to be valid", v)
		}
	}
	for _, v := range invalid {
		if grouperPathPattern.MatchString(v) {
			t.Errorf("expected %q to be invalid", v)
		}
	}
}
