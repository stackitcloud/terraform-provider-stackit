variable "project_id" {}
variable "name" {}
variable "policy" {}

resource "stackit_affinity_group" "affinity_group" {
  project_id = var.project_id
  name       = var.name
  policy     = var.policy
}