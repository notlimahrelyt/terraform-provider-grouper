package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestStemDataSource_Schema(t *testing.T) {
	t.Parallel()
	var resp datasource.SchemaResponse
	NewStemDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("stem data source schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "name", "display_name", "description", "parent_stem_name"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing stem data source attribute %q", name)
		}
	}
	if _, ok := NewStemDataSource().(datasource.DataSourceWithConfigValidators); !ok {
		t.Error("stem data source must enforce exactly-one-of via config validators")
	}
}

func TestGroupDataSource_Schema(t *testing.T) {
	t.Parallel()
	var resp datasource.SchemaResponse
	NewGroupDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("group data source schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "name", "display_name", "description", "stem_name", "enabled", "type_of_group"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing group data source attribute %q", name)
		}
	}
	if _, ok := NewGroupDataSource().(datasource.DataSourceWithConfigValidators); !ok {
		t.Error("group data source must enforce exactly-one-of via config validators")
	}
}

func TestSubjectDataSource_Schema(t *testing.T) {
	t.Parallel()
	var resp datasource.SchemaResponse
	NewSubjectDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("subject data source schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"subject_id", "subject_source_id", "subject_attribute_names", "id", "name", "source_id", "attributes"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing subject data source attribute %q", name)
		}
	}
	if !resp.Schema.Attributes["subject_id"].IsRequired() {
		t.Error("subject_id must be required")
	}
}
