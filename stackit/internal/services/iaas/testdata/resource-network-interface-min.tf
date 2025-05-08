variable "project_id" {}
variable "name" {}

resource "stackit_network" "network" {
  project_id = var.project_id
  name       = var.name
}

resource "stackit_network_interface" "network_interface" {
  project_id = var.project_id
  network_id = stackit_network.network.network_id
}

resource "stackit_public_ip" "public_ip" {
  project_id = var.project_id
}