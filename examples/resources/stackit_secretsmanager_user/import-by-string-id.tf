# Only use the import statement, if you want to import an existing secretsmanager user
import {
  to = stackit_secretsmanager_user.import-example
  id = "${var.project_id},${var.secret_instance_id},${var.secret_user_id}"
}
