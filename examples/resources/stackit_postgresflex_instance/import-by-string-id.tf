# Only use the import statement, if you want to import an existing postgresflex instance
import {
  to = stackit_postgresflex_instance.import-example
  id = "${var.project_id},${var.region},${var.postgres_instance_id}"
}
