# Only use the import statement, if you want to import an existing observability http check
import {
  to = stackit_observability_http_check.import-example
  id = "${var.project_id},${var.observability_instance_id},${var.observability_http_check_id}"
}
