---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_postgresflex_database Resource - stackit"
subcategory: ""
description: |-
  Postgres Flex database resource schema. Must have a region specified in the provider configuration.
---

# stackit_postgresflex_database (Resource)

Postgres Flex database resource schema. Must have a `region` specified in the provider configuration.

## Example Usage

```terraform
resource "stackit_postgresflex_database" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "mydb"
  owner       = "myusername"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `instance_id` (String) ID of the Postgres Flex instance.
- `name` (String) Database name.
- `owner` (String) Username of the database owner.
- `project_id` (String) STACKIT project ID to which the instance is associated.

### Read-Only

- `database_id` (String) Database ID.
- `id` (String) Terraform's internal resource ID. It is structured as "`project_id`,`instance_id`,`database_id`".