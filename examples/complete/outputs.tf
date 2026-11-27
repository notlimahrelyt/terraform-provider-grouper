output "stem_name" {
  description = "Full name of the application folder."
  value       = grouper_stem.app.name
}

output "admins_group" {
  description = "Name and UUID of the admins group."
  value = {
    name = grouper_group.admins.name
    id   = grouper_group.admins.id
  }
}

output "users_group" {
  description = "Name and UUID of the users group."
  value = {
    name = grouper_group.users.name
    id   = grouper_group.users.id
  }
}
