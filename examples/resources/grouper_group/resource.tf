resource "grouper_stem" "cloud" {
  name         = "example:apps:myteam:example"
  display_name = "Example"
}

resource "grouper_group" "aws_admins" {
  name         = "${grouper_stem.cloud.name}:aws-admins"
  display_name = "AWS Administrators"
  description  = "Administrators of centrally managed AWS services"
}

# Import by UUID:
#   terraform import grouper_group.aws_admins 09b488181c7d42ba9eb880fd7ad698d3
#
# Import by full name:
#   terraform import grouper_group.aws_admins name:example:apps:myteam:example:aws-admins
