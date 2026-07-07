# Only use the import statement, if you want to import an existing federated identity provider
import {
  to = stackit_service_account_federated_identity_provider.import-example
  id = "${var.project_id},${var.service_account_email},${var.federation_id}"
}
