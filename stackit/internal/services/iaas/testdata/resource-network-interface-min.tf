variable "parent_container_id" {}
variable "name" {}
variable "owner_email" {}
variable "network_name" {}
variable "ipv4_prefix" {}
variable "ipv4_nameserver_0" {}
variable "ipv4_nameserver_1" {}

resource "stackit_resourcemanager_project" "example" {
  parent_container_id = var.parent_container_id
  name                = var.name
  owner_email         = var.owner_email
}

resource "stackit_network" "network" {
  project_id       = stackit_resourcemanager_project.example.project_id
  name             = var.network_name
  ipv4_prefix      = var.ipv4_prefix
  ipv4_nameservers = [var.ipv4_nameserver_0, var.ipv4_nameserver_1]
}

resource "stackit_network_interface" "network_interface" {
  project_id = stackit_resourcemanager_project.example.project_id
  network_id = stackit_network.network.network_id
}

resource "stackit_public_ip" "public_ip" {
  project_id = stackit_resourcemanager_project.example.project_id
}