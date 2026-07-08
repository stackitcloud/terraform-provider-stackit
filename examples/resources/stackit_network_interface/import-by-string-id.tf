# Only use the import statement, if you want to import an existing network interface
import {
  to = stackit_network_interface.import-example
  id = "${var.project_id},${var.region},${var.network_id},${var.network_interface_id}"
}
