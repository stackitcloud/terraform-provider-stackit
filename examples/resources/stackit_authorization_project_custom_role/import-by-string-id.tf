# Only use the import statement, if you want to import an existing custom role
import {
  to = stackit_authorization_project_custom_role.import-example
  id = "${var.project_id},${var.custom_role_id}"
}
