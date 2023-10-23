---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_secretsmanager_user Data Source - stackit"
subcategory: ""
description: |-
  Secrets Manager user data source schema. Must have a region specified in the provider configuration.
---

# stackit_secretsmanager_user (Data Source)

Secrets Manager user data source schema. Must have a `region` specified in the provider configuration.

## Example Usage

```terraform
data "stackit_secretsmanager_user" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  user_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `instance_id` (String) ID of the Secrets Manager instance.
- `project_id` (String) STACKIT Project ID to which the instance is associated.
- `user_id` (String) The user's ID.

### Read-Only

- `description` (String) A user chosen description to differentiate between multiple users. Can't be changed after creation.
- `id` (String) Terraform's internal data source identifier. It is structured as "`project_id`,`instance_id`,`user_id`".
- `username` (String) An auto-generated user name.
- `write_enabled` (Boolean) If true, the user has writeaccess to the secrets engine.