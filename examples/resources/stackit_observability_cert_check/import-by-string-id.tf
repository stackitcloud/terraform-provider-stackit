# Only use the import statement, if you want to import an existing observability cert check
import {
  to = stackit_observability_cert_check.import-example
  id = "${var.project_id},${var.observability_instance_id},${var.observability_cert_check_id}"
}
