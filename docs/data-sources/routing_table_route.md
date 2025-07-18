---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_routing_table_route Data Source - stackit"
subcategory: ""
description: |-
  Routing table route datasource schema. Must have a region specified in the provider configuration.
  ~> This datasource is part of the routing-tables experiment and is likely going to undergo significant changes or be removed in the future. Use it at your own discretion.
---

# stackit_routing_table_route (Data Source)

Routing table route datasource schema. Must have a `region` specified in the provider configuration.

~> This datasource is part of the routing-tables experiment and is likely going to undergo significant changes or be removed in the future. Use it at your own discretion.

## Example Usage

```terraform
data "stackit_routing_table_route" "example" {
  organization_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_area_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  routing_table_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  route_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `network_area_id` (String) The network area ID to which the routing table is associated.
- `organization_id` (String) STACKIT organization ID to which the routing table is associated.
- `route_id` (String) Route ID.
- `routing_table_id` (String) The routing tables ID.

### Optional

- `region` (String) The resource region. If not defined, the provider region is used.

### Read-Only

- `created_at` (String) Date-time when the route was created
- `destination` (Attributes) Destination of the route. (see [below for nested schema](#nestedatt--destination))
- `id` (String) Terraform's internal datasource ID. It is structured as "`organization_id`,`region`,`network_area_id`,`routing_table_id`,`route_id`".
- `labels` (Map of String) Labels are key-value string pairs which can be attached to a resource container
- `next_hop` (Attributes) Next hop destination. (see [below for nested schema](#nestedatt--next_hop))
- `updated_at` (String) Date-time when the route was updated

<a id="nestedatt--destination"></a>
### Nested Schema for `destination`

Read-Only:

- `type` (String) CIDRV type. Possible values are: `cidrv4`, `cidrv6`. Only `cidrv4` is supported during experimental stage.
- `value` (String) An CIDR string.


<a id="nestedatt--next_hop"></a>
### Nested Schema for `next_hop`

Read-Only:

- `type` (String) Possible values are: `blackhole`, `internet`, `ipv4`, `ipv6`. Only `cidrv4` is supported during experimental stage..
- `value` (String) Either IPv4 or IPv6 (not set for blackhole and internet). Only IPv4 supported during experimental stage.
