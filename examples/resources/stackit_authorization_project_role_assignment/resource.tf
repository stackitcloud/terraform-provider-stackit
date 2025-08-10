resource "stackit_authorization_project_role_assignment" "example" {
  resource_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  role        = "owner"
  subject     = "john.doe@stackit.cloud"
}

# Only use the import statement, if you want to import an existing project role assignment
import {
  to = stackit_authorization_project_role_assignment.import-example
  id = "${var.project_id},${var.project_role_assignment_role},${var.project_role_assignment_subject}"
}
