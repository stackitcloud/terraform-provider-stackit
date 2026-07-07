# Only use the import statement, if you want to import an existing network area region
import {
  to = stackit_network_area_region.import-example
  id = "${var.organization_id},${var.network_area_id},${var.region}"
}
