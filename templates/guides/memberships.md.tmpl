---
page_title: "Understanding memberships - Grouper Provider"
subcategory: ""
description: |-
  Direct vs effective vs composite membership, and how the provider manages them.
---

# Understanding memberships

Grouper distinguishes several kinds of membership. This provider manages exactly
one kind, deliberately.

## Membership types

- **Direct (immediate)** — the subject is explicitly listed as a member of the
  group. **This is the only type `grouper_membership` manages.** Reads use
  `memberFilter=Immediate` so nothing else is mistaken for a managed membership.
- **Effective** — the subject is a member because of a nested group (a group that
  is itself a member of this group). Not managed here.
- **Composite** — membership derived from group math (union, intersection,
  complement) on other groups. Not managed here.
- **Nested group membership** — a group can be a subject/member of another group.
  You can manage that edge with `grouper_membership` by using the nested group as
  the subject, because Grouper treats a group as a valid subject.
- **External / Loader-managed** — memberships maintained by the Grouper Loader or
  another system. The provider never touches these.

## One resource per membership

`grouper_membership` manages a single direct membership. There is intentionally
no authoritative "manage all members of this group" resource, because such a
resource could remove members added manually, by the Loader, or by another
Terraform state. Every add uses `replaceAllExisting = "F"`, and every delete
removes only the one named subject.

To manage a set of members, use `for_each`:

```hcl
resource "grouper_membership" "admins" {
  for_each   = toset(["abc123", "xyz456"])
  group_name = grouper_group.aws_admins.name
  subject_id = each.value
}
```

Members outside that set are left untouched.

## Subjects and identifiers

`subject_id` accepts either the subject's canonical id or a login identifier such
as a username or NetID; Grouper resolves either. After apply, `subject_source_id` is
populated with the subject's resolved source. Use the `grouper_subject` data
source if you need the canonical id.

## Do not double-manage

Never manage the same Grouper object (stem, group, or membership) from more than
one Terraform state or module. Two states fighting over the same object will
produce perpetual diffs and can undo each other's changes.
