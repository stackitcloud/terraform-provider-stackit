---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_postgresflex_database Data Source - stackit"
subcategory: ""
description: |-
  Postgres Flex database resource schema. Must have a region specified in the provider configuration.
---

# stackit_postgresflex_database (Data Source)

Postgres Flex database resource schema. Must have a `region` specified in the provider configuration.

## Example Usage

```terraform
data "stackit_postgresflex_database" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  database_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `database_id` (String) Database ID.
- `instance_id` (String) ID of the Postgres Flex instance.
- `project_id` (String) STACKIT project ID to which the instance is associated.

### Read-Only

- `id` (String) Terraform's internal resource ID. It is structured as "`project_id`,`instance_id`,`database_id`".
- `name` (String) Database name.
- `owner` (String) Username of the database owner.