variable "project_id" {}
variable "name" {}

resource "stackit_containerregistry_artifactory" "this" {
  project_id = var.project_id
  name       = var.name
}
