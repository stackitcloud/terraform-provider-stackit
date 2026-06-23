variable "project_id" {}

resource "stackit_sfs_project_lock" "project_lock" {
  project_id = var.project_id
}
