# Only use the import statement, if you want to import an existing postgresflex database
import {
  to = stackit_postgresflex_database.import-example
  id = "${var.project_id},${var.region},${var.postgres_instance_id},${var.postgres_database_id}"
}
