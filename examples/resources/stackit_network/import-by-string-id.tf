# Only use the import statement, if you want to import an existing network
# Note: There will be a conflict which needs to be resolved manually.
# These attributes cannot be configured together: [ipv4_prefix,ipv4_prefix_length,ipv4_gateway]
import {
  to = stackit_network.import-example
  id = "${var.project_id},${var.region},${var.network_id}"
}
