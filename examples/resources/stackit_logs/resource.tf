resource "stackit_logs_instance" "git" {
  project_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region         = "eu01"
  display_name   = "logs-instance-example"
  retention_days = 30
}

resource "stackit_logs_instance" "logs" {
  project_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region         = "eu01"
  display_name   = "logs-instance-example"
  retention_days = 30
  acl = [
    "0.0.0.0/0"
  ]
  description = "Example description"
}

# Only use the import statement, if you want to import an existing git resource
import {
  to = stackit_logs_instance.import-example
  id = "${var.project_id},${var.region},${var.logs_instance_id}"
}
