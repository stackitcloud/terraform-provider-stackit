# Only use the import statement, if you want to import an existing valkey credential
import {
  to = stackit_valkey_credential.import-example
  id = "${var.project_id},${var.region},${var.valkey_instance_id},${var.valkey_credential_id}"
}
