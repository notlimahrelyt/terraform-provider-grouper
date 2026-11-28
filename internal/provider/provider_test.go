package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	fwschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestProvider_Metadata(t *testing.T) {
	t.Parallel()
	p := New("test1.2.3")()
	var resp provider.MetadataResponse
	p.Metadata(context.Background(), provider.MetadataRequest{}, &resp)
	if resp.TypeName != "grouper" {
		t.Errorf("TypeName = %q, want grouper", resp.TypeName)
	}
	if resp.Version != "test1.2.3" {
		t.Errorf("Version = %q, want test1.2.3", resp.Version)
	}
}

func TestProvider_Schema(t *testing.T) {
	t.Parallel()
	p := New("test")()
	var resp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema produced diagnostics: %v", resp.Diagnostics)
	}

	want := []string{"base_url", "ws_version", "username", "password", "request_timeout", "insecure_skip_verify"}
	for _, name := range want {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing provider attribute %q", name)
		}
	}

	// Every attribute must have a description.
	for name, attr := range resp.Schema.Attributes {
		if attr.GetDescription() == "" && attr.GetMarkdownDescription() == "" {
			t.Errorf("attribute %q has no description", name)
		}
	}
}

func TestProvider_PasswordIsSensitive(t *testing.T) {
	t.Parallel()
	p := New("test")()
	var resp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)

	pw, ok := resp.Schema.Attributes["password"].(fwschema.StringAttribute)
	if !ok {
		t.Fatalf("password attribute is not a StringAttribute")
	}
	if !pw.IsSensitive() {
		t.Error("password attribute must be marked sensitive")
	}

	// base_url must NOT be sensitive (sanity check on the assertion above).
	bu := resp.Schema.Attributes["base_url"].(fwschema.StringAttribute)
	if bu.IsSensitive() {
		t.Error("base_url should not be sensitive")
	}
}

func TestProvider_RegistersResources(t *testing.T) {
	t.Parallel()
	p := New("test")()
	ctors := p.Resources(context.Background())

	got := map[string]bool{}
	for _, ctor := range ctors {
		var meta resource.MetadataResponse
		ctor().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "grouper"}, &meta)
		got[meta.TypeName] = true
	}
	for _, want := range []string{"grouper_stem", "grouper_group", "grouper_membership"} {
		if !got[want] {
			t.Errorf("resource %q not registered (have %v)", want, got)
		}
	}

	got = map[string]bool{}
	for _, ctor := range p.DataSources(context.Background()) {
		var meta datasource.MetadataResponse
		ctor().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "grouper"}, &meta)
		got[meta.TypeName] = true
	}
	for _, want := range []string{"grouper_stem", "grouper_group", "grouper_subject"} {
		if !got[want] {
			t.Errorf("data source %q not registered (have %v)", want, got)
		}
	}
}
