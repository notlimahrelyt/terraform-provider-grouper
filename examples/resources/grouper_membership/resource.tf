resource "grouper_group" "aws_admins" {
  name         = "example:apps:myteam:example:aws-admins"
  display_name = "AWS Administrators"
}

# One resource per direct membership. Use for_each to manage a set of members
# without affecting anyone added to the group by other means.
resource "grouper_membership" "aws_admins" {
  for_each = toset([
    "abc123",
    "xyz456",
  ])

  group_name = grouper_group.aws_admins.name
  subject_id = each.value
}

# Import format: <group_name>/<subject_source_id>/<subject_id>
#   terraform import 'grouper_membership.aws_admins["abc123"]' \
#     'example:apps:myteam:example:aws-admins/ldap/abc123'
#
# If the source is unknown/omitted, leave it empty:
#   .../example:aws-admins//abc123
