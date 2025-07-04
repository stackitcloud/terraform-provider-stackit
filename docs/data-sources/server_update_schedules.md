---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_server_update_schedules Data Source - stackit"
subcategory: ""
description: |-
  Server update schedules datasource schema. Must have a region specified in the provider configuration.
  ~> This datasource is in beta and may be subject to breaking changes in the future. Use with caution. See our guide https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs/guides/opting_into_beta_resources for how to opt-in to use beta resources.
---

# stackit_server_update_schedules (Data Source)

Server update schedules datasource schema. Must have a `region` specified in the provider configuration.

~> This datasource is in beta and may be subject to breaking changes in the future. Use with caution. See our [guide](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs/guides/opting_into_beta_resources) for how to opt-in to use beta resources.

## Example Usage

```terraform
data "stackit_server_update_schedules" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `project_id` (String) STACKIT Project ID (UUID) to which the server is associated.
- `server_id` (String) Server ID (UUID) to which the update schedule is associated.

### Optional

- `region` (String) The resource region. If not defined, the provider region is used.

### Read-Only

- `id` (String) Terraform's internal data source identifier. It is structured as "`project_id`,`region`,`server_id`".
- `items` (Attributes List) (see [below for nested schema](#nestedatt--items))

<a id="nestedatt--items"></a>
### Nested Schema for `items`

Read-Only:

- `enabled` (Boolean) Is the update schedule enabled or disabled.
- `maintenance_window` (Number) Maintenance window [1..24].
- `name` (String) The update schedule name.
- `rrule` (String) Update schedule described in `rrule` (recurrence rule) format.
- `update_schedule_id` (Number)
