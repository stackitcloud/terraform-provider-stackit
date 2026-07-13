# Only use the import statement, if you want to import an existing loadbalancer
import {
  to = stackit_loadbalancer.import-example
  id = "${var.project_id},${var.region},${var.loadbalancer_name}"
}
