# Only use the import statement, if you want to import an existing logme instance
import {
  to = stackit_logme_instance.import-example
  id = "${var.project_id},${var.logme_instance_id}"
}
