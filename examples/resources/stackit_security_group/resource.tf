resource "stackit_security_group" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "my_security_group"
  labels = {
    "key" = "value"
  }
}

# Only use the import statement, if you want to import an existing security group
import {
  to = stackit_security_group.import-example
  id = "${var.project_id},${var.security_group_id}"
}