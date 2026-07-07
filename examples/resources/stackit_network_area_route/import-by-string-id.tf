# Only use the import statement, if you want to import an existing network area route
import {
  to = stackit_network_area_route.import-example
  id = "${var.organization_id},${var.network_area_id},${var.region},${var.network_area_route_id}"
}
