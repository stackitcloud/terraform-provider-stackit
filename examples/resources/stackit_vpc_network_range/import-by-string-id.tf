# Only use the import statement, if you want to import an existing dns zone
import {
  to = stackit_vpc_network_range.example
  id = "${var.project_id},${var.vpc_id},${var.region},${var.vpc_network_range_id}"
}