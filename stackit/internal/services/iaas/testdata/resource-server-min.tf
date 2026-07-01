variable "server_name" {}
variable "network_name" {}
variable "machine_type" {}
variable "image_id" {}
variable "parent_container_id" {}
variable "name" {}
variable "owner_email" {}

resource "stackit_resourcemanager_project" "example" {
  parent_container_id = var.parent_container_id
  name                = var.name
  owner_email         = var.owner_email
}

resource "stackit_network" "network" {
  project_id = stackit_resourcemanager_project.example.project_id
  name       = var.network_name
}

resource "stackit_network_interface" "nic" {
  project_id = stackit_resourcemanager_project.example.project_id
  network_id = stackit_network.network.network_id
}

resource "stackit_server" "server" {
  project_id   = stackit_resourcemanager_project.example.project_id
  name         = var.server_name
  machine_type = var.machine_type
  boot_volume = {
    source_type           = "image"
    size                  = 16
    source_id             = var.image_id
    delete_on_termination = true
  }
  network_interfaces = [
    stackit_network_interface.nic.network_interface_id
  ]
}
