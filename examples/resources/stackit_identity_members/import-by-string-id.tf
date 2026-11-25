# Only use the import statement if you want to import existing Identity Group Members
import {
  to = stackit_identity_group_members.members
  id = "${var.organization_id},${var.group_id}"
}