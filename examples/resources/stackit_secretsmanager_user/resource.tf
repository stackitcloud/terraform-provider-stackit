resource "stackit_secretsmanager_user" "example" {
  project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  description   = "Example user"
  write_enabled = false
}

# Only use the import statement, if you want to import an existing secretsmanager user
import {
  to = stackit_secretsmanager_user.import-example
  id = "${var.project_id},${var.secret_instance_id},${var.secret_user_id}"
}
