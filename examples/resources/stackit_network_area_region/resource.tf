resource "stackit_network_area_region" "example" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_area_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  ipv4 = {
    transfer_network = "10.1.2.0/24"
    network_ranges = [
      {
        prefix = "10.0.0.0/16"
      }
    ]
  }
}

# Only use the import statement, if you want to import an existing network area region
import {
  to = stackit_network_area_region.import-example
  id = "${var.organization_id},${var.network_area_id},${var.region}"
}
