
variable "project_id" {}
variable "name" {}
variable "rules" {}

resource "stackit_sfs_export_policy" "exportpolicy" {
  project_id = var.project_id
  name       = var.name
  rules      = var.rules
}
