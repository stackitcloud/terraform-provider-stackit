# Only use the import statement, if you want to import an existing ske cluster
import {
  to = stackit_ske_cluster.import-example
  id = "${var.project_id},${var.region},${var.ske_name}"
}
