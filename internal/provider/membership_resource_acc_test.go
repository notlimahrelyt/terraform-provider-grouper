package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccMembershipResource exercises the direct-membership lifecycle against a
// real Grouper deployment. Requires TF_ACC, the GROUPER_* env vars, and a
// resolvable GROUPER_TEST_SUBJECT_ID (a NetID or subject id). It creates a group
// under the test stem, adds the subject, imports, and deletes.
func TestAccMembershipResource(t *testing.T) {
	requireTestSubject(t)
	group := testChildStem("tfacc-mgroup")
	subject := testSubjectID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMembershipConfig(group, subject),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("grouper_membership.m", "group_name", group),
					resource.TestCheckResourceAttr("grouper_membership.m", "subject_id", subject),
					resource.TestCheckResourceAttr("grouper_membership.m", "membership_type", "immediate"),
					resource.TestCheckResourceAttrSet("grouper_membership.m", "id"),
					resource.TestCheckResourceAttrSet("grouper_membership.m", "subject_source_id"),
				),
			},
			{
				// Import using the deterministic id (the default import id).
				ResourceName:      "grouper_membership.m",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccMembershipResource_preservesUnrelated verifies that removing one
// Terraform-managed membership does not remove other members of the same group.
// Requires a second resolvable subject via GROUPER_TEST_SUBJECT_ID_2.
func TestAccMembershipResource_preservesUnrelated(t *testing.T) {
	requireTestSubject(t)
	if testSubjectID2() == "" {
		t.Skip("GROUPER_TEST_SUBJECT_ID_2 not set; skipping unrelated-member preservation test")
	}
	group := testChildStem("tfacc-mpreserve")
	s1, s2 := testSubjectID(), testSubjectID2()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Both members present.
				Config: testAccTwoMembershipsConfig(group, s1, s2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("grouper_membership.m1", "id"),
					resource.TestCheckResourceAttrSet("grouper_membership.m2", "id"),
				),
			},
			{
				// Remove m2. If deleting m2 wrongly removed m1, m1's refresh would
				// drop it from state and this step would show a non-empty plan.
				Config: testAccMembershipConfigNamed(group, "m1", s1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("grouper_membership.m1", "subject_id", s1),
					resource.TestCheckResourceAttrSet("grouper_membership.m1", "id"),
				),
			},
		},
	})
}

func testAccMembershipConfig(group, subject string) string {
	return testAccMembershipConfigNamed(group, "m", subject)
}

func testAccMembershipConfigNamed(group, resName, subject string) string {
	src := ""
	if testSubjectSourceID() != "" {
		src = fmt.Sprintf("  subject_source_id = %q\n", testSubjectSourceID())
	}
	return fmt.Sprintf(`
provider "grouper" {}

resource "grouper_group" "g" {
  name         = %[1]q
  display_name = "Membership Test Group"
}

resource "grouper_membership" %[2]q {
  group_name = grouper_group.g.name
  subject_id = %[3]q
%[4]s}
`, group, resName, subject, src)
}

func testAccTwoMembershipsConfig(group, s1, s2 string) string {
	src := ""
	if testSubjectSourceID() != "" {
		src = fmt.Sprintf("  subject_source_id = %q\n", testSubjectSourceID())
	}
	return fmt.Sprintf(`
provider "grouper" {}

resource "grouper_group" "g" {
  name         = %[1]q
  display_name = "Membership Test Group"
}

resource "grouper_membership" "m1" {
  group_name = grouper_group.g.name
  subject_id = %[2]q
%[4]s}

resource "grouper_membership" "m2" {
  group_name = grouper_group.g.name
  subject_id = %[3]q
%[4]s}
`, group, s1, s2, src)
}
