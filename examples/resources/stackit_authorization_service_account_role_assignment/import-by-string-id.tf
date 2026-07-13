# Only use the import statement, if you want to import an existing service account assignment
import {
  to = stackit_authorization_service_account_assignment.sa
  id = "${var.resource_id},${var.service_account_assignment_role},${var.service_account_assignment_subject}"
}
