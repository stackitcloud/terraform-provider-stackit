# Only use the import statement, if you want to import an existing security group
import {
  to = stackit_security_group.import-example
  id = "${var.project_id},${var.region},${var.security_group_id}"
}
