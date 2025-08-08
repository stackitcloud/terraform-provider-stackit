resource "stackit_routing_table_route" "example" {
  organization_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_area_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  routing_table_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  destination = {
    type  = "cidrv4"
    value = "192.168.178.0/24"
  }
  next_hop = {
    type  = "ipv4"
    value = "192.168.178.1"
  }
  labels = {
    "key" = "value"
  }
}

# Only use the import statement, if you want to import an existing routing table route
import {
  to = stackit_routing_table_route.import-example
  id = "${var.organization_id},${var.region},${var.network_area_id},${var.routing_table_id},${var.routing_table_route_id}"
}