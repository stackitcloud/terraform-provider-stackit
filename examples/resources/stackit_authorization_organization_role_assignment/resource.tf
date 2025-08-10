resource "stackit_authorization_organization_role_assignment" "example" {
  resource_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  role        = "owner"
  subject     = "john.doe@stackit.cloud"
}

# Only use the import statement, if you want to import an existing organization role assignment
import {
  to = stackit_authorization_organization_role_assignment.import-example
  id = "${var.organization_id},${var.org_role_assignment_role},${var.org_role_assignment_subject}"
}
