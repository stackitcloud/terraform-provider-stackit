variable "parent_container_id" {}
variable "owner_email" {}
variable "project_name" {}

variable "name" {}
variable "description" {}
variable "label_key" {}
variable "label_value" {}

# no test candidate, just needed for the testing setup
resource "stackit_resourcemanager_project" "project" {
  parent_container_id = var.parent_container_id
  name                = var.project_name
  labels = {
    "enable-vpc" = "true"
  }
  owner_email = var.owner_email
}

resource "stackit_vpc" "vpc" {
  project_id  = stackit_resourcemanager_project.project.project_id
  name        = var.name
  description = var.description
  labels = {
    (var.label_key) = var.label_value
  }
}
