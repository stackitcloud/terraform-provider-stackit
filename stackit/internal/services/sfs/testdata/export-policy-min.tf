
variable "project_id" {}
variable "name" {}

resource "stackit_sfs_export_policy" "exportpolicy" {
  project_id = var.project_id
  name       = var.name
}
