---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_mongodbflex_instance Data Source - stackit"
subcategory: ""
description: |-
  MongoDB Flex instance data source schema. Must have a region specified in the provider configuration.
---

# stackit_mongodbflex_instance (Data Source)

MongoDB Flex instance data source schema. Must have a `region` specified in the provider configuration.

## Example Usage

```terraform
data "stackit_mongodbflex_instance" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `instance_id` (String) ID of the MongoDB Flex instance.
- `project_id` (String) STACKIT project ID to which the instance is associated.

### Optional

- `region` (String) The resource region. If not defined, the provider region is used.

### Read-Only

- `acl` (List of String) The Access Control List (ACL) for the MongoDB Flex instance.
- `backup_schedule` (String) The backup schedule. Should follow the cron scheduling system format (e.g. "0 0 * * *").
- `flavor` (Attributes) (see [below for nested schema](#nestedatt--flavor))
- `id` (String) Terraform's internal data source ID. It is structured as "`project_id`,`region`,`instance_id`".
- `name` (String) Instance name.
- `options` (Attributes) Custom parameters for the MongoDB Flex instance. (see [below for nested schema](#nestedatt--options))
- `replicas` (Number)
- `storage` (Attributes) (see [below for nested schema](#nestedatt--storage))
- `version` (String)

<a id="nestedatt--flavor"></a>
### Nested Schema for `flavor`

Read-Only:

- `cpu` (Number)
- `description` (String)
- `id` (String)
- `ram` (Number)


<a id="nestedatt--options"></a>
### Nested Schema for `options`

Read-Only:

- `daily_snapshot_retention_days` (Number) The number of days that daily backups will be retained.
- `monthly_snapshot_retention_months` (Number) The number of months that monthly backups will be retained.
- `point_in_time_window_hours` (Number) The number of hours back in time the point-in-time recovery feature will be able to recover.
- `snapshot_retention_days` (Number) The number of days that continuous backups (controlled via the `backup_schedule`) will be retained.
- `type` (String) Type of the MongoDB Flex instance.
- `weekly_snapshot_retention_weeks` (Number) The number of weeks that weekly backups will be retained.


<a id="nestedatt--storage"></a>
### Nested Schema for `storage`

Read-Only:

- `class` (String)
- `size` (Number)
