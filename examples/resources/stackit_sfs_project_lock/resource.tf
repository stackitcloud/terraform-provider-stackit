resource "stackit_sfs_project_lock" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# Only use the import statement, if you want to import an existing project lock
import {
  to = stackit_sfs_project_lock.example
  id = "${var.project_id},${var.region}"
}