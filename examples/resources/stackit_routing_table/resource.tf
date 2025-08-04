resource "stackit_routing_table" "example" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_area_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example"
  labels = {
    "key" = "value"
  }
}

# Only use the import statement, if you want to import an existing routing table
import {
  to = stackit_routing_table.import-example
  id = "${var.organization_id},${var.region},${var.network_area_id},${var.routing_table_id}"
}
