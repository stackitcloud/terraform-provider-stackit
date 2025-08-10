resource "stackit_sqlserverflex_user" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  username    = "username"
  roles       = ["role"]
}

# Only use the import statement, if you want to import an existing sqlserverflex user
import {
  to = stackit_sqlserverflex_user.import-example
  id = "${var.project_id},${var.region},${var.sql_instance_id},${var.sql_user_id}"
}