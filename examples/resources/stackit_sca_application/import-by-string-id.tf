# Only use the import statement, if you want to import an existing application
import {
  to = stackit_sca_application.import-example
  id = "${var.project_id},${var.region},${var.environment_id},${var.application_id}"
}
