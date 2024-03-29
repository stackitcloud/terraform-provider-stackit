---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_secretsmanager_user Resource - stackit"
subcategory: ""
description: |-
  Secrets Manager user resource schema. Must have a region specified in the provider configuration.
---

# stackit_secretsmanager_user (Resource)

Secrets Manager user resource schema. Must have a `region` specified in the provider configuration.

## Example Usage

```terraform
resource "stackit_secretsmanager_user" "example" {
  project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  description   = "Example user"
  write_enabled = false
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `description` (String) A user chosen description to differentiate between multiple users. Can't be changed after creation.
- `instance_id` (String) ID of the Secrets Manager instance.
- `project_id` (String) STACKIT Project ID to which the instance is associated.
- `write_enabled` (Boolean) If true, the user has writeaccess to the secrets engine.

### Read-Only

- `id` (String) Terraform's internal resource identifier. It is structured as "`project_id`,`instance_id`,`user_id`".
- `password` (String, Sensitive) An auto-generated password.
- `user_id` (String) The user's ID.
- `username` (String) An auto-generated user name.
