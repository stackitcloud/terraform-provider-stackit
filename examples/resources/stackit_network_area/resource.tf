resource "stackit_network_area" "example" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example-network-area"
  network_ranges = [
    {
      prefix = "192.168.0.0/24"
    }
  ]
  transfer_network = "192.168.1.0/24"
  labels = {
    "key" = "value"
  }
}

# Only use the import statement, if you want to import an existing network area
import {
  to = stackit_network_area.import-example
  id = "${var.organization_id},${var.network_area_id}"
}