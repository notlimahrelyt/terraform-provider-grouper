# Import by Grouper UUID:
terraform import grouper_group.aws_admins 09b488181c7d42ba9eb880fd7ad698d3

# Import by full group name (resolved to its UUID):
terraform import grouper_group.aws_admins name:example:apps:myteam:aws-admins
