# Only use the import statement, if you want to import an existing service account
import {
  to = stackit_service_account.import-example
  id = "${var.project_id},${var.service_account_email}"
}
