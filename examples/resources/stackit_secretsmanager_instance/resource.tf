resource "stackit_secretsmanager_instance" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-instance"
  acls       = ["XXX.XXX.XXX.X/XX", "XX.XXX.XX.X/XX"]
}

# Only use the import statement, if you want to import an existing secretsmanager instance
import {
  to = stackit_secretsmanager_instance.import-example
  id = "${var.project_id},${var.secret_instance_id}"
}