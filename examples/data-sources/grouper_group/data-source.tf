data "grouper_group" "admins" {
  name = "example:apps:myteam:aws-admins"
}

output "group_uuid" {
  value = data.grouper_group.admins.id
}
