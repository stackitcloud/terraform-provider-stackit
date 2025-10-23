variable "project_id" {}
variable "name" {}

resource "stackit_network" "network" {
  project_id = var.project_id
  name       = var.name
}