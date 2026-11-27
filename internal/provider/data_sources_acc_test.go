package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStemDataSource(t *testing.T) {
	name := testChildStem("tfacc-stemds")
	config := fmt.Sprintf(`
provider "grouper" {}

resource "grouper_stem" "s" {
  name         = %[1]q
  display_name = "DS Test Stem"
}

data "grouper_stem" "by_name" {
  name = grouper_stem.s.name
}
`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.grouper_stem.by_name", "name", name),
				resource.TestCheckResourceAttrSet("data.grouper_stem.by_name", "id"),
				resource.TestCheckResourceAttr("data.grouper_stem.by_name", "display_name", "DS Test Stem"),
				resource.TestCheckResourceAttrPair("data.grouper_stem.by_name", "id", "grouper_stem.s", "id"),
			),
		}},
	})
}

func TestAccGroupDataSource(t *testing.T) {
	name := testChildStem("tfacc-groupds")
	config := fmt.Sprintf(`
provider "grouper" {}

resource "grouper_group" "g" {
  name         = %[1]q
  display_name = "DS Test Group"
}

data "grouper_group" "by_name" {
  name = grouper_group.g.name
}
`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.grouper_group.by_name", "name", name),
				resource.TestCheckResourceAttrSet("data.grouper_group.by_name", "id"),
				resource.TestCheckResourceAttr("data.grouper_group.by_name", "enabled", "true"),
				resource.TestCheckResourceAttrPair("data.grouper_group.by_name", "id", "grouper_group.g", "id"),
			),
		}},
	})
}

func TestAccSubjectDataSource(t *testing.T) {
	requireTestSubject(t)
	subject := testSubjectID()
	config := fmt.Sprintf(`
provider "grouper" {}

data "grouper_subject" "person" {
  subject_id = %[1]q
}
`, subject)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.grouper_subject.person", "subject_id", subject),
				resource.TestCheckResourceAttrSet("data.grouper_subject.person", "id"),
				resource.TestCheckResourceAttrSet("data.grouper_subject.person", "source_id"),
			),
		}},
	})
}
