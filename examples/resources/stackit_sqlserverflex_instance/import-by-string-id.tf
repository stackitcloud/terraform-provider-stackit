# Only use the import statement, if you want to import an existing sqlserverflex instance
import {
  to = stackit_sqlserverflex_instance.import-example
  id = "${var.project_id},${var.region},${var.sql_instance_id}"
}
