# Only use the import statement, if you want to import an existing mongodbflex user
import {
  to = stackit_mongodbflex_user.import-example
  id = "${var.project_id},${var.region},${var.instance_id},${user_id}"
}
