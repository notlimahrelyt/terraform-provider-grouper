package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func membershipSchema(t *testing.T) fwresource.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewMembershipResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("membership schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestMembershipResource_Schema(t *testing.T) {
	t.Parallel()
	s := membershipSchema(t)

	for _, name := range []string{"id", "group_name", "subject_id", "subject_source_id", "membership_type"} {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing membership attribute %q", name)
			continue
		}
		if attr.GetDescription() == "" {
			t.Errorf("attribute %q has no description", name)
		}
	}
	if !s.Attributes["group_name"].IsRequired() {
		t.Error("group_name must be required")
	}
	if !s.Attributes["subject_id"].IsRequired() {
		t.Error("subject_id must be required")
	}
}

func TestMembershipResource_ImplementsImport(t *testing.T) {
	t.Parallel()
	if _, ok := NewMembershipResource().(resource.ResourceWithImportState); !ok {
		t.Error("membership resource must implement ResourceWithImportState")
	}
}

func TestMembershipID_RoundTrip(t *testing.T) {
	t.Parallel()
	cases := []struct {
		group, source, subject string
	}{
		{"example:apps:myteam:test:admins", "ldap", "abc123"},
		{"example:g", "", "abc123"},     // empty source
		{"example:g", "ldap", "cn=a/b"}, // subject id containing the separator
	}
	for _, tc := range cases {
		id := buildMembershipID(tc.group, tc.source, tc.subject)
		g, s, subj, ok := parseMembershipID(id)
		if !ok {
			t.Errorf("parseMembershipID(%q) not ok", id)
			continue
		}
		if g != tc.group || s != tc.source || subj != tc.subject {
			t.Errorf("round-trip %q => (%q,%q,%q), want (%q,%q,%q)", id, g, s, subj, tc.group, tc.source, tc.subject)
		}
	}
}

func TestParseMembershipID_Invalid(t *testing.T) {
	t.Parallel()
	invalid := []string{
		"onlyone",
		"group/source", // missing subject
		"/ldap/abc",    // empty group
		"group/ldap/",  // empty subject
	}
	for _, id := range invalid {
		if _, _, _, ok := parseMembershipID(id); ok {
			t.Errorf("expected %q to be invalid", id)
		}
	}
}
