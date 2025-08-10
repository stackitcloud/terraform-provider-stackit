resource "stackit_network_area_route" "example" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_area_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  prefix          = "192.168.0.0/24"
  next_hop        = "192.168.0.0"
  labels = {
    "key" = "value"
  }
}

# Only use the import statement, if you want to import an existing network area route
import {
  to = stackit_network_area_route.import-example
  id = "${var.organization_id},${var.network_area_id},${var.network_area_route_id}"
}