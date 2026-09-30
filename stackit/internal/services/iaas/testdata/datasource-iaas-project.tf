variable "organization_id" {}
variable "parent_container_id" {}
variable "name" {}
variable "owner_email" {}

# no test candidate, just needed for the testing setup
resource "stackit_resourcemanager_project" "with_public_network" {
  parent_container_id = var.parent_container_id
  name                = var.name
  owner_email         = var.owner_email
}

# no test candidate, just needed for the testing setup
resource "stackit_network_area" "network_area" {
  organization_id = var.organization_id
  name            = var.name
}

# no test candidate, just needed for the testing setup
resource "stackit_network_area_region" "network_area_region" {
  organization_id = var.organization_id
  network_area_id = stackit_network_area.network_area.network_area_id
  ipv4 = {
    network_ranges = [
      {
        prefix = "10.0.0.0/16"
      },
      {
        prefix = "10.2.2.0/24"
      }
    ]
    transfer_network = "10.1.2.0/24"
  }
}

# no test candidate, just needed for the testing setup
resource "stackit_resourcemanager_project" "with_network_area" {
  parent_container_id = stackit_network_area.network_area.organization_id
  name                = var.name
  labels = {
    "networkArea" = stackit_network_area.network_area.network_area_id
  }
  owner_email = var.owner_email
  depends_on = [stackit_network_area_region.network_area_region]
}


data "stackit_iaas_project" "public" {
  project_id = stackit_resourcemanager_project.with_public_network.project_id
}

data "stackit_iaas_project" "sna" {
  project_id = stackit_resourcemanager_project.with_network_area.project_id
}
