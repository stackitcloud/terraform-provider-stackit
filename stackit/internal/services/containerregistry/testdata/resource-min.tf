variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

resource "stackit_containerregistry_artifactory" "this" {
  project_id = var.project_id
  name       = var.name
}