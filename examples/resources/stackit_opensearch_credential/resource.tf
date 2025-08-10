resource "stackit_opensearch_credential" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# Only use the import statement, if you want to import an existing opensearch credential
import {
  to = stackit_opensearch_credential.import-example
  id = "${var.project_id},${var.instance_id},${var.credential_id}"
}