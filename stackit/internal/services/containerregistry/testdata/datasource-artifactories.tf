variable "project_id" {}

data "stackit_containerregistry_artifactories" "this" {
  project_id = var.project_id
}
