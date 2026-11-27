# Look up a subject by login identifier (e.g. a username or NetID) or subject id; Grouper resolves either.
data "grouper_subject" "person" {
  subject_id = "abc123"
  # subject_source_id = "sourceId" # optional, to disambiguate

  # Request extra attributes (opt-in; values land in `attributes`).
  # Attribute names depend on how subject sources are configured in your Grouper.
  subject_attribute_names = ["email", "description"]
}

output "canonical_subject_id" {
  value = data.grouper_subject.person.id
}

output "email" {
  value = data.grouper_subject.person.attributes["email"]
}
