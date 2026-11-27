package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccGroupResource exercises the full group lifecycle against a real Grouper
// deployment. Gated by TF_ACC and the GROUPER_* environment variables. The
// parent GROUPER_TEST_STEM must already exist.
func TestAccGroupResource(t *testing.T) {
	stem := testStem()
	name := testChildStem("tfacc-group")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupConfig(name, "Acc Test Group", "created by acceptance test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("grouper_group.test", "name", name),
					resource.TestCheckResourceAttr("grouper_group.test", "display_name", "Acc Test Group"),
					resource.TestCheckResourceAttr("grouper_group.test", "description", "created by acceptance test"),
					resource.TestCheckResourceAttrSet("grouper_group.test", "id"),
					resource.TestCheckResourceAttr("grouper_group.test", "stem_name", stem),
				),
			},
			{
				Config: testAccGroupConfig(name, "Acc Test Group Renamed", "updated by acceptance test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("grouper_group.test", "display_name", "Acc Test Group Renamed"),
					resource.TestCheckResourceAttr("grouper_group.test", "description", "updated by acceptance test"),
				),
			},
			{
				ResourceName:      "grouper_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grouper_group.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return "name:" + name, nil
				},
			},
		},
	})
}

func testAccGroupConfig(name, displayName, description string) string {
	return fmt.Sprintf(`
provider "grouper" {}

resource "grouper_group" "test" {
  name         = %[1]q
  display_name = %[2]q
  description  = %[3]q
}
`, name, displayName, description)
}
