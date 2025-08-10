resource "stackit_loadbalancer_observability_credential" "example" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  display_name = "example-credentials"
  username     = "example-user"
  password     = "example-password"
}

# Only use the import statement, if you want to import an existing loadbalancer observability credential
import {
  to = stackit_loadbalancer_observability_credential.import-example
  id = "${var.project_id},${var.region},${var.credentials_ref}"
}
