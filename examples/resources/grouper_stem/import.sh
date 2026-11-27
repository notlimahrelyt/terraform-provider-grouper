# Import by Grouper UUID:
terraform import grouper_stem.example 0123456789abcdef0123456789abcdef

# Import by full stem name (resolved to its UUID):
terraform import grouper_stem.example name:example:apps:myteam:example
