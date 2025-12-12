resource "stackit_server_network_interface_attach" "attached_network_interface" {
  project_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id            = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_interface_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# Only use the import statement, if you want to import an existing server network interface attachment
import {
  to = stackit_server_network_interface_attach.import-example
  id = "${var.project_id},${var.region},${var.server_id},${var.network_interface_id}"
}