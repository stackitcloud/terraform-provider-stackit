# Only use the import statement, if you want to import an existing loadbalancer observability credential
import {
  to = stackit_loadbalancer_observability_credential.import-example
  id = "${var.project_id},${var.region},${var.credentials_ref}"
}