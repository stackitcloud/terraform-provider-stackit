# Only use the import statement, if you want to import an existing server service account attachment
import {
  to = stackit_server_service_account_attach.import-example
  id = "${var.project_id},${var.region},${var.server_id},${var.service_account_email}"
}
