# Only use the import statement if you want to import an existing Identity Group
import {
  to = stackit_identity_group.import-example
  id = "${var.organization_id},${var.group_id}"
}