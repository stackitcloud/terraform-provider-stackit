# Only use the import statement, if you want to import an existing scf organization
import {
  to = stackit_scf_organization.import-example
  id = "${var.project_id},${var.region},${var.org_id}"
}
