resource "stackit_authorization_folder_role_assignment" "example" {
  resource_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  role        = "owner"
  subject     = "john.doe@stackit.cloud"
}

# Only use the import statement, if you want to import an existing folder role assignment
import {
  to = stackit_authorization_folder_role_assignment.import-example
  id = "${var.folder_id},${var.folder_role_assignment_role},${var.folder_role_assignment_subject}"
}
