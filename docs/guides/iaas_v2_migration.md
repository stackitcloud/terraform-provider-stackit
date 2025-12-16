---
page_title: "Migration of IaaS resources from versions <= v0.73.0"
---

# Migration of IaaS resources from versions <= v0.73.0

The release of the STACKIT IaaS API v2 provides a lot of new features, but also includes some breaking changes 
(when coming from v1 of the STACKIT IaaS API) which must be somehow represented on Terraform side. Please use the 
guide below to migrate your resources properly.

## Breaking change: Network area route resource (stackit_network_area_route)

The `stackit_network_area_route` resource did undergo some changes. See the example below how to migrate your resources.

**Configuration for <= v0.73.0**

```terraform
resource "stackit_network_area_route" "example" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_area_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  prefix          = "192.168.0.0/24" # prefix field got removed for provider versions > v0.73.0, use the new destination field instead
  next_hop        = "192.168.0.0" # schema of the next_hop field changed, see below
}
```

**Configuration for > v0.73.0**

```terraform
resource "stackit_network_area_route" "example" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_area_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  destination = { # the new 'destination' field replaces the old 'prefix' field
    type  = "cidrv4"
    value = "192.168.0.0/24" # migration: put the value of the old 'prefix' field here
  }
  next_hop = {
    type  = "ipv4"
    value = "192.168.0.0" # migration: put the value of the old 'next_hop' field here
  }
}
```

## Breaking change: Network area route resource (stackit_network_area_route)
