package provider

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Lifecycle tests run each resource and data source through real Terraform
// plans and applies against fakeGrouper. They need a Terraform binary but no
// Grouper server, so they run in CI on every change. Each step also asserts
// that a plan immediately after apply is empty.

func TestLifecycleStem(t *testing.T) {
	f := newFakeGrouper(t)
	const name = "example:apps:inventory"

	config := func(display, description string) string {
		return f.providerConfig() + fmt.Sprintf(`
resource "grouper_stem" "app" {
  name         = %q
  display_name = %q
  description  = %q
}
`, name, display, description)
	}

	var firstID string
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             f.checkStemGone(name),
		Steps: []resource.TestStep{
			{
				Config: config("Inventory", "Inventory groups"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("grouper_stem.app", "name", name),
					resource.TestCheckResourceAttr("grouper_stem.app", "display_name", "Inventory"),
					resource.TestCheckResourceAttr("grouper_stem.app", "description", "Inventory groups"),
					resource.TestCheckResourceAttr("grouper_stem.app", "parent_stem_name", "example:apps"),
					resource.TestCheckResourceAttrWith("grouper_stem.app", "id", func(id string) error {
						firstID = id
						return expectEqual("stem UUID", f.stem(name).UUID, id)
					}),
				),
			},
			{
				// Mutable fields update in place.
				Config: config("Inventory System", "Groups for the inventory system"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("grouper_stem.app", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("grouper_stem.app", "id", func(id string) error {
						return expectEqual("stem UUID after update", firstID, id)
					}),
					func(*terraform.State) error {
						s := f.stem(name)
						if err := expectEqual("display extension", "Inventory System", s.DisplayExtension); err != nil {
							return err
						}
						return expectEqual("description", "Groups for the inventory system", s.Description)
					},
				),
			},
			{
				ResourceName:      "grouper_stem.app",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grouper_stem.app",
				ImportState:       true,
				ImportStateId:     "name:" + name,
				ImportStateVerify: true,
			},
			{
				// Deleted outside Terraform: refresh drops it and apply recreates it.
				PreConfig: func() { f.deleteStem(name) },
				Config:    config("Inventory System", "Groups for the inventory system"),
				Check: func(*terraform.State) error {
					if f.stem(name) == nil {
						return fmt.Errorf("stem %q was not recreated", name)
					}
					return nil
				},
			},
		},
	})
}

func TestLifecycleStem_renameForcesReplacement(t *testing.T) {
	f := newFakeGrouper(t)
	config := func(name string) string {
		return f.providerConfig() + fmt.Sprintf(`
resource "grouper_stem" "app" {
  name = %q
}
`, name)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("example:apps:old"),
				Check:  resource.TestCheckResourceAttr("grouper_stem.app", "display_name", "old"),
			},
			{
				Config: config("example:apps:new"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("grouper_stem.app", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: func(*terraform.State) error {
					if f.stem("example:apps:old") != nil {
						return fmt.Errorf("old stem still exists after replacement")
					}
					if f.stem("example:apps:new") == nil {
						return fmt.Errorf("new stem was not created")
					}
					return nil
				},
			},
		},
	})
}

func TestLifecycleStem_invalidName(t *testing.T) {
	f := newFakeGrouper(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: f.providerConfig() + `
resource "grouper_stem" "bad" {
  name = "example::apps"
}
`,
				ExpectError: regexp.MustCompile(`colon-delimited Grouper path`),
			},
		},
	})
}

func TestLifecycleGroup(t *testing.T) {
	f := newFakeGrouper(t)
	const name = "example:apps:admins"

	config := func(display, description string) string {
		return f.providerConfig() + fmt.Sprintf(`
resource "grouper_group" "admins" {
  name         = %q
  display_name = %q
  description  = %q
}
`, name, display, description)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             f.checkGroupGone(name),
		Steps: []resource.TestStep{
			{
				Config: config("Admins", "Administrators"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("grouper_group.admins", "name", name),
					resource.TestCheckResourceAttr("grouper_group.admins", "display_name", "Admins"),
					resource.TestCheckResourceAttr("grouper_group.admins", "description", "Administrators"),
					resource.TestCheckResourceAttr("grouper_group.admins", "stem_name", "example:apps"),
					resource.TestCheckResourceAttrWith("grouper_group.admins", "id", func(id string) error {
						return expectEqual("group UUID", f.group(name).UUID, id)
					}),
				),
			},
			{
				Config: config("Application Admins", "People who administer the application"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("grouper_group.admins", plancheck.ResourceActionUpdate),
					},
				},
				Check: func(*terraform.State) error {
					g := f.group(name)
					if err := expectEqual("display extension", "Application Admins", g.DisplayExtension); err != nil {
						return err
					}
					return expectEqual("description", "People who administer the application", g.Description)
				},
			},
			{
				ResourceName:      "grouper_group.admins",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grouper_group.admins",
				ImportState:       true,
				ImportStateId:     "name:" + name,
				ImportStateVerify: true,
			},
			{
				PreConfig: func() { f.deleteGroup(name) },
				Config:    config("Application Admins", "People who administer the application"),
				Check: func(*terraform.State) error {
					if f.group(name) == nil {
						return fmt.Errorf("group %q was not recreated", name)
					}
					return nil
				},
			},
		},
	})
}

func TestLifecycleMembership(t *testing.T) {
	f := newFakeGrouper(t)
	const group = "example:apps:users"

	// One named resource per member. (The import test step cannot address
	// for_each instances, so this avoids for_each.)
	config := func(members ...string) string {
		c := f.providerConfig() + fmt.Sprintf(`
resource "grouper_group" "users" {
  name = %q
}
`, group)
		for _, m := range members {
			c += fmt.Sprintf(`
resource "grouper_membership" %[1]q {
  group_name = grouper_group.users.name
  subject_id = %[1]q
}
`, m)
		}
		return c
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             f.checkGroupGone(group),
		Steps: []resource.TestStep{
			{
				Config: config("abc123", "xyz456"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("grouper_membership.abc123", "id", group+"/ldap/abc123"),
					resource.TestCheckResourceAttr("grouper_membership.abc123", "subject_source_id", "ldap"),
					resource.TestCheckResourceAttr("grouper_membership.abc123", "membership_type", "immediate"),
					f.checkMembers(group, "10000001", "10000002"),
				),
			},
			{
				// Someone adds ext789 by hand, and xyz456 is removed from the
				// configuration. Only xyz456 is removed; ext789 is left alone.
				PreConfig: func() { f.addMember(group, "ext789") },
				Config:    config("abc123"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("grouper_membership.xyz456", plancheck.ResourceActionDestroy),
						plancheck.ExpectResourceAction("grouper_membership.abc123", plancheck.ResourceActionNoop),
					},
				},
				Check: f.checkMembers(group, "10000001", "10000003"),
			},
			{
				ResourceName:      "grouper_membership.abc123",
				ImportState:       true,
				ImportStateId:     group + "/ldap/abc123",
				ImportStateVerify: true,
			},
			{
				// Removed outside Terraform: refresh notices and apply re-adds it.
				PreConfig: func() { f.removeMember(group, "abc123") },
				Config:    config("abc123"),
				Check:     f.checkMembers(group, "10000001", "10000003"),
			},
		},
	})

	// Destroying the configuration removed only the managed member.
	if got := f.memberIDs(group); len(got) != 0 {
		t.Errorf("group was deleted, but fake still lists members %v", got)
	}
}

func TestLifecycleMembership_adoptsExisting(t *testing.T) {
	f := newFakeGrouper(t)
	const group = "example:apps:users"
	groupOnly := f.providerConfig() + fmt.Sprintf(`
resource "grouper_group" "users" {
  name = %q
}
`, group)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: groupOnly},
			{
				// abc123 is already a member; creating the resource adopts it
				// instead of failing.
				PreConfig: func() { f.addMember(group, "abc123") },
				Config: groupOnly + `
resource "grouper_membership" "m" {
  group_name = grouper_group.users.name
  subject_id = "abc123"
}
`,
				Check: f.checkMembers(group, "10000001"),
			},
		},
	})
}

func TestLifecycleMembership_subjectChangeForcesReplacement(t *testing.T) {
	f := newFakeGrouper(t)
	const group = "example:apps:users"
	config := func(subject string) string {
		return f.providerConfig() + fmt.Sprintf(`
resource "grouper_group" "users" {
  name = %q
}

resource "grouper_membership" "m" {
  group_name = grouper_group.users.name
  subject_id = %q
}
`, group, subject)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("abc123"), Check: f.checkMembers(group, "10000001")},
			{
				Config: config("xyz456"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("grouper_membership.m", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: f.checkMembers(group, "10000002"),
			},
		},
	})
}

func TestLifecycleMembership_nestedGroup(t *testing.T) {
	f := newFakeGrouper(t)
	config := f.providerConfig() + `
resource "grouper_group" "outer" {
  name = "example:apps:all-staff"
}

resource "grouper_group" "inner" {
  name = "example:apps:it-staff"
}

resource "grouper_membership" "nested" {
  group_name = grouper_group.outer.name
  subject_id = grouper_group.inner.id
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("grouper_membership.nested", "subject_id", "grouper_group.inner", "id"),
					resource.TestCheckResourceAttr("grouper_membership.nested", "subject_source_id", "g:gsa"),
					func(*terraform.State) error {
						return f.checkMembers("example:apps:all-staff", f.group("example:apps:it-staff").UUID)(nil)
					},
				),
			},
		},
	})
}

func TestLifecycleMembership_unknownSubject(t *testing.T) {
	f := newFakeGrouper(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: f.providerConfig() + `
resource "grouper_group" "users" {
  name = "example:apps:users"
}

resource "grouper_membership" "m" {
  group_name = grouper_group.users.name
  subject_id = "nobody"
}
`,
				ExpectError: regexp.MustCompile(`could not resolve subject "nobody"`),
			},
		},
	})
}

func TestLifecycleDataSources(t *testing.T) {
	f := newFakeGrouper(t)
	f.addStem("example:apps:existing", "Existing App", "Created outside Terraform")
	stemID := f.stem("example:apps:existing").UUID

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: f.providerConfig() + fmt.Sprintf(`
data "grouper_stem" "by_name" {
  name = "example:apps:existing"
}

data "grouper_stem" "by_id" {
  id = %q
}

resource "grouper_group" "g" {
  name         = "example:apps:existing:readers"
  display_name = "Readers"
}

data "grouper_group" "by_name" {
  name = grouper_group.g.name
}

data "grouper_group" "by_id" {
  id = grouper_group.g.id
}

data "grouper_subject" "person" {
  subject_id              = "abc123"
  subject_attribute_names = ["email"]
}
`, stemID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.grouper_stem.by_name", "id", stemID),
					resource.TestCheckResourceAttr("data.grouper_stem.by_name", "display_name", "Existing App"),
					resource.TestCheckResourceAttr("data.grouper_stem.by_name", "description", "Created outside Terraform"),
					resource.TestCheckResourceAttr("data.grouper_stem.by_name", "parent_stem_name", "example:apps"),
					resource.TestCheckResourceAttr("data.grouper_stem.by_id", "name", "example:apps:existing"),

					resource.TestCheckResourceAttrPair("data.grouper_group.by_name", "id", "grouper_group.g", "id"),
					resource.TestCheckResourceAttr("data.grouper_group.by_name", "display_name", "Readers"),
					resource.TestCheckResourceAttr("data.grouper_group.by_name", "enabled", "true"),
					resource.TestCheckResourceAttr("data.grouper_group.by_id", "name", "example:apps:existing:readers"),

					resource.TestCheckResourceAttr("data.grouper_subject.person", "id", "10000001"),
					resource.TestCheckResourceAttr("data.grouper_subject.person", "name", "Ada Example"),
					resource.TestCheckResourceAttr("data.grouper_subject.person", "source_id", "ldap"),
					resource.TestCheckResourceAttr("data.grouper_subject.person", "attributes.email", "ada@example.edu"),
				),
			},
		},
	})
}

func TestLifecycleDataSources_errors(t *testing.T) {
	f := newFakeGrouper(t)
	cases := map[string]struct {
		config string
		err    string
	}{
		"stem needs exactly one lookup": {
			config: `
data "grouper_stem" "s" {
  id   = "abc"
  name = "example:apps"
}
`,
			err: `Invalid Attribute Combination`,
		},
		"missing stem": {
			config: `
data "grouper_stem" "s" {
  name = "example:apps:missing"
}
`,
			err: `not found`,
		},
		"missing group": {
			config: `
data "grouper_group" "g" {
  name = "example:apps:missing"
}
`,
			err: `not found`,
		},
		"missing subject": {
			config: `
data "grouper_subject" "s" {
  subject_id = "nobody"
}
`,
			err: `Unable to read Grouper subject`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: f.providerConfig() + tc.config, ExpectError: regexp.MustCompile(tc.err)},
				},
			})
		})
	}
}

func TestLifecycleProvider_configErrors(t *testing.T) {
	f := newFakeGrouper(t)
	t.Setenv("GROUPER_BASE_URL", "")
	t.Setenv("GROUPER_USERNAME", "")
	t.Setenv("GROUPER_PASSWORD", "")
	t.Setenv("GROUPER_REQUEST_TIMEOUT", "")

	stemData := `
data "grouper_stem" "s" {
  name = "example:apps"
}
`
	cases := map[string]struct {
		provider string
		err      string
	}{
		"missing base_url": {
			provider: `provider "grouper" {}`,
			err:      `Missing Grouper base URL`,
		},
		"bad timeout": {
			provider: fmt.Sprintf(`
provider "grouper" {
  base_url        = %q
  request_timeout = "soon"
}
`, f.server.URL),
			err: `Invalid request_timeout`,
		},
		"wrong password": {
			provider: fmt.Sprintf(`
provider "grouper" {
  base_url = %q
  username = %q
  password = "wrong"
}
`, f.server.URL+"/grouper-ws/servicesRest", fakeUser),
			err: `authentication failed`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: tc.provider + stemData, ExpectError: regexp.MustCompile(tc.err)},
				},
			})
		})
	}
}

func TestLifecycleProvider_environment(t *testing.T) {
	f := newFakeGrouper(t)
	t.Setenv("GROUPER_BASE_URL", f.server.URL+"/grouper-ws/servicesRest/")
	t.Setenv("GROUPER_USERNAME", fakeUser)
	t.Setenv("GROUPER_PASSWORD", fakePassword)
	t.Setenv("GROUPER_REQUEST_TIMEOUT", "10s")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Everything comes from the environment, including a base URL
				// with a trailing slash.
				Config: `
provider "grouper" {}

data "grouper_stem" "s" {
  name = "example:apps"
}
`,
				Check: resource.TestCheckResourceAttr("data.grouper_stem.s", "name", "example:apps"),
			},
		},
	})
}

// --- helpers ---

func (f *fakeGrouper) checkStemGone(name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if f.stem(name) != nil {
			return fmt.Errorf("stem %q still exists after destroy", name)
		}
		return nil
	}
}

func (f *fakeGrouper) checkGroupGone(name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if f.group(name) != nil {
			return fmt.Errorf("group %q still exists after destroy", name)
		}
		return nil
	}
}

// checkMembers asserts the exact set of direct member subject ids of a group.
func (f *fakeGrouper) checkMembers(group string, wantIDs ...string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got := f.memberIDs(group)
		want := append([]string(nil), wantIDs...)
		sort.Strings(want)
		if len(got) == 0 && len(want) == 0 {
			return nil
		}
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("members of %q: got %v, want %v", group, got, want)
		}
		return nil
	}
}

func expectEqual(what, want, got string) error {
	if want != got {
		return fmt.Errorf("%s: got %q, want %q", what, got, want)
	}
	return nil
}
