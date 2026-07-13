# Only use the import statement, if you want to import an existing public ip
import {
  to = stackit_public_ip.import-example
  id = "${var.project_id},${var.region},${var.public_ip_id}"
}
