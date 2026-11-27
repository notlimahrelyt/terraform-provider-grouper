resource "grouper_stem" "cloud" {
  name         = "example:apps:myteam:example"
  display_name = "Example"
  description  = "Example folder managed by Terraform"
}

# Import by UUID:
#   terraform import grouper_stem.cloud 0123456789abcdef0123456789abcdef
#
# Import by full name:
#   terraform import grouper_stem.cloud name:example:apps:myteam:example
