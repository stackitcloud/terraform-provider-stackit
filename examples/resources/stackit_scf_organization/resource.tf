resource "stackit_scf_organization" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example"
}

resource "stackit_scf_organization" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "example"
  platform_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  quota_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  suspended   = false
}

# Only use the import statement, if you want to import an existing scf organization
import {
  to = stackit_scf_organization.import-example
  id = "${var.project_id},${var.region},${var.org_id}"
}
