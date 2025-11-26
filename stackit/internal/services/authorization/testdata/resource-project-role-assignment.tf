variable "name" {}
variable "role" {}
variable "owner_email" {}
variable "subject" {}
variable "parent_container_id" {}

resource "stackit_resourcemanager_project" "project" {
  name                = var.name
  owner_email         = var.owner_email
  parent_container_id = var.parent_container_id
}

resource "stackit_authorization_project_role_assignment" "pra" {
  resource_id = stackit_resourcemanager_project.project.project_id
  role        = var.role
  subject     = var.owner_email
}