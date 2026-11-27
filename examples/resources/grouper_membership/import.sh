# Import format: <group_name>/<subject_source_id>/<subject_id>
terraform import grouper_membership.example 'example:apps:myteam:aws-admins/sourceId/abc123'

# The source component may be empty if unknown:
terraform import grouper_membership.example 'example:apps:myteam:aws-admins//abc123'
