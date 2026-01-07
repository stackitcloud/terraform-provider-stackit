resource "stackit_sfs_export_policy" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example"
  rules = [
    {
      ip_acl = ["172.16.0.0/24", "172.16.0.250/32"]
      order  = 1
    }
  ]
}

# Only use the import statement, if you want to import an existing export policy
import {
  to = stackit_sfs_export_policy.example
  id = "${var.project_id},${var.region},${var.policy_id}"
}