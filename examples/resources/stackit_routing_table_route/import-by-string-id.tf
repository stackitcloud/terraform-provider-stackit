# Only use the import statement, if you want to import an existing routing table route
import {
  to = stackit_routing_table_route.import-example
  id = "${var.organization_id},${var.region},${var.network_area_id},${var.routing_table_id},${var.routing_table_route_id}"
}
