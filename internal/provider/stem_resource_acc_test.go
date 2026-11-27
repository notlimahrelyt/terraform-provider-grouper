package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccStemResource exercises the full stem lifecycle against a real Grouper
// deployment. It only runs when TF_ACC is set and the GROUPER_* environment
// variables (including a test-namespace GROUPER_TEST_STEM) are provided.
//
// The parent GROUPER_TEST_STEM must already exist and the service account must
// have privileges to create child stems within it.
func TestAccStemResource(t *testing.T) {
	name := testChildStem("tfacc-stem")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create + Read.
			{
				Config: testAccStemConfig(name, "Acc Test Stem", "created by acceptance test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("grouper_stem.test", "name", name),
					resource.TestCheckResourceAttr("grouper_stem.test", "display_name", "Acc Test Stem"),
					resource.TestCheckResourceAttr("grouper_stem.test", "description", "created by acceptance test"),
					resource.TestCheckResourceAttrSet("grouper_stem.test", "id"),
					resource.TestCheckResourceAttr("grouper_stem.test", "parent_stem_name", testStem()),
				),
			},
			// Update mutable metadata.
			{
				Config: testAccStemConfig(name, "Acc Test Stem Renamed", "updated by acceptance test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("grouper_stem.test", "display_name", "Acc Test Stem Renamed"),
					resource.TestCheckResourceAttr("grouper_stem.test", "description", "updated by acceptance test"),
				),
			},
			// Import by UUID.
			{
				ResourceName:      "grouper_stem.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Import by name.
			{
				ResourceName:      "grouper_stem.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return "name:" + name, nil
				},
			},
		},
	})
}

func testAccStemConfig(name, displayName, description string) string {
	return fmt.Sprintf(`
provider "grouper" {}

resource "grouper_stem" "test" {
  name         = %[1]q
  display_name = %[2]q
  description  = %[3]q
}
`, name, displayName, description)
}
