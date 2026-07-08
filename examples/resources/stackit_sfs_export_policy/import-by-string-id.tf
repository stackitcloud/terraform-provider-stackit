# Only use the import statement, if you want to import an existing export policy
import {
  to = stackit_sfs_export_policy.example
  id = "${var.project_id},${var.region},${var.policy_id}"
}
