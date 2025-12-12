resource "stackit_server_service_account_attach" "attached_service_account" {
  project_id            = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  service_account_email = "service-account@stackit.cloud"
}

# Only use the import statement, if you want to import an existing server service account attachment
import {
  to = stackit_server_service_account_attach.import-example
  id = "${var.project_id},${var.region},${var.server_id},${var.service_account_email}"
}