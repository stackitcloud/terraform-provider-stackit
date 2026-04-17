
variable "project_id" {}
variable "region" {}
variable "name" {}
variable "rules" {}

resource "stackit_sfs_export_policy" "exportpolicy" {
  project_id = var.project_id
  region     = var.region
  name       = var.name
  rules      = var.rules
}
