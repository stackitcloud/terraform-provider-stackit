# Only use the import statement, if you want to import an existing static route
import {
  to = stackit_vpc_routing_table_static_route.import-example
  id = "${var.project_id},${var.vpc_id},${var.region},${var.routing_table_id},${var.route_id}"
}
