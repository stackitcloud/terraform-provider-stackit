# Only use the import statement, if you want to import an existing routing table
import {
  to = stackit_vpc_routing_table.import-example
  id = "${var.project_id},${var.vpc_id},${var.region},${var.routing_table_id}"
}
