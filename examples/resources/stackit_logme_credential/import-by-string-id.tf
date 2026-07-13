# Only use the import statement, if you want to import an existing logme credential
import {
  to = stackit_logme_credential.import-example
  id = "${var.project_id},${var.logme_instance_id},${var.logme_credentials_id}"
}
