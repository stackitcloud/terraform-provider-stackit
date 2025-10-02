resource "stackit_scf_organization_manager" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  org_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# Only use the import statement, if you want to import an existing scf org user
# The password field is still null after import and must be entered manually in the state.
import {
  to = stackit_scf_organization_manager.import-example
  id = "${var.project_id},${var.region},${var.org_id},${var.user_id}"
}