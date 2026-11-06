variable "project_id" {
  type = string
}

data "stackit_containerregistry_artifactories" "this" {
  project_id = var.project_id
}