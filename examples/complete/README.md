# Complete example: application groups

Creates the folder and groups one application needs, and fills them with
members:

```text
example:apps:inventory          folder  "Inventory"
example:apps:inventory:admins   group   "Inventory Admins"
example:apps:inventory:users    group   "Inventory Users"
```

Each member is a separate `grouper_membership`, so adding or removing a name
from `admins` or `users` changes only that one membership. Members added to these
groups outside Terraform are left alone.

## Prerequisites

- The parent folder (`parent_stem`) already exists.
- The Grouper account has `STEM_ADMIN` and `CREATE` on the parent folder. See the
  [authentication guide](../../docs/guides/authentication.md).

## Usage

```shell
export GROUPER_BASE_URL="https://grouper.example.edu/grouper-ws/servicesRest"
export GROUPER_USERNAME="terraform-svc"
export GROUPER_PASSWORD="..."

cp terraform.tfvars.example terraform.tfvars   # then edit it
terraform init
terraform plan
terraform apply
```

## Inputs

| Name          | Description                                              | Type          | Default |
| ------------- | -------------------------------------------------------- | ------------- | ------- |
| `parent_stem` | Existing folder to create the application folder under   | `string`      |         |
| `app_id`      | Short system name; the last component of the folder name | `string`      |         |
| `app_name`    | Human-readable name used in display names                | `string`      |         |
| `admins`      | Subject ids or login identifiers for the admins group    | `set(string)` | `[]`    |
| `users`       | Subject ids or login identifiers for the users group     | `set(string)` | `[]`    |

## Outputs

| Name           | Description                         |
| -------------- | ----------------------------------- |
| `stem_name`    | Full name of the application folder |
| `admins_group` | Name and UUID of the admins group   |
| `users_group`  | Name and UUID of the users group    |
