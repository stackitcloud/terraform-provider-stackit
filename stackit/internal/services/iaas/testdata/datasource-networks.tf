variable "organization_id" {}
variable "service_account_mail" {}
variable "project_name" {}
variable "network_name" {}

# bootstrap a new project so we will have a reproducible example from the network list endpoint
resource "stackit_resourcemanager_project" "project" {
  parent_container_id = var.organization_id
  name                = var.project_name
  owner_email         = var.service_account_mail
}

# bootstrap a network in the new project
resource "stackit_network" "network" {
  project_id = stackit_resourcemanager_project.project.project_id
  name       = var.network_name
}

# the actual datasource we want to test
data "stackit_networks" "networks" {
  project_id = stackit_resourcemanager_project.project.project_id
  depends_on = [stackit_network.network]
}
