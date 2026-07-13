# Only use the import statement, if you want to import an existing public ip associate
import {
  to = stackit_public_ip_associate.import-example
  id = "${var.project_id},${var.region},${var.public_ip_id},${var.network_interface_id}"
}
