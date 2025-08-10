resource "stackit_service_account" "sa" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "sa01"
}

# Only use the import statement, if you want to import an existing service account
import {
  to = stackit_service_account.import-example
  id = "${var.project_id},${var.service_account_email}"
}