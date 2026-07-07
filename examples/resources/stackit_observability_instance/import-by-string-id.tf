# Only use the import statement, if you want to import an existing observability instance
import {
  to = stackit_observability_instance.import-example
  id = "${var.project_id},${var.observability_instance_id}"
}
