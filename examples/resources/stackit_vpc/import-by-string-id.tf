# Only use the import statement, if you want to import an existing vpc
import {
  to = stackit_vpc.import-example
  id = "${var.project_id},${var.vpc_id}"
}
