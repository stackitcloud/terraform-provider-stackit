# Only use the import statement, if you want to import an existing region
import {
  to = stackit_vpc_region.import-example
  id = "${var.project_id},${var.vpc_id},${var_region}"
}
