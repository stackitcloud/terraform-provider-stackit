variable "project_id" {}
variable "name" {}

variable "parent_container_id" {}
variable "project_name" {}
variable "owner_email" {}

resource "stackit_resourcemanager_project" "example" {
  parent_container_id = var.parent_container_id
  name                = var.project_name
  owner_email         = var.owner_email
}

resource "stackit_network" "network" {
  project_id = stackit_resourcemanager_project.example.project_id
  name       = var.name
}