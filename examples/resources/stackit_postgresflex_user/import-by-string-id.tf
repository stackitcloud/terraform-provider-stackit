# Only use the import statement, if you want to import an existing postgresflex user
import {
  to = stackit_postgresflex_user.import-example
  id = "${var.project_id},${var.region},${var.postgres_instance_id},${var.user_id}"
}
