# Only use the import statement, if you want to import an existing VPN connection
import {
  to = stackit_vpn_connection.example
  id = "${var.project_id},${var.region},${var.gateway_id},${var.connection_id}"
}
