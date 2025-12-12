resource "stackit_network_interface" "example" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  allowed_addresses  = ["192.168.0.0/24"]
  security_group_ids = ["xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"]
}

# Only use the import statement, if you want to import an existing network interface
import {
  to = stackit_network_interface.import-example
  id = "${var.project_id},${var.region},${var.network_id},${var.network_interface_id}"
}