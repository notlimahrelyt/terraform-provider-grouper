# Builds a standard layout for one application:
#
#   <parent_stem>:<app_id>          folder
#   <parent_stem>:<app_id>:admins   group
#   <parent_stem>:<app_id>:users    group

resource "grouper_stem" "app" {
  name         = "${var.parent_stem}:${var.app_id}"
  display_name = var.app_name
  description  = "Groups for ${var.app_name}. Managed by Terraform."
}

resource "grouper_group" "admins" {
  name         = "${grouper_stem.app.name}:admins"
  display_name = "${var.app_name} Admins"
  description  = "Administrators of ${var.app_name}. Managed by Terraform."
}

resource "grouper_group" "users" {
  name         = "${grouper_stem.app.name}:users"
  display_name = "${var.app_name} Users"
  description  = "Users of ${var.app_name}. Managed by Terraform."
}

resource "grouper_membership" "admins" {
  for_each = var.admins

  group_name = grouper_group.admins.name
  subject_id = each.value
}

resource "grouper_membership" "users" {
  for_each = var.users

  group_name = grouper_group.users.name
  subject_id = each.value
}
