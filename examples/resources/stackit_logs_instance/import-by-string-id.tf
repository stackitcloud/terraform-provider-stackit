# Only use the import statement, if you want to import an existing logs instance
import {
  to = stackit_logs_instance.import-example
  id = "${var.project_id},${var.region},${var.logs_instance_id}"
}
