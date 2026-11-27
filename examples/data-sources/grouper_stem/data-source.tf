# Look up a stem by name...
data "grouper_stem" "by_name" {
  name = "example:apps:myteam"
}

# ...or by UUID.
data "grouper_stem" "by_id" {
  id = "0123456789abcdef0123456789abcdef"
}

output "stem_uuid" {
  value = data.grouper_stem.by_name.id
}
