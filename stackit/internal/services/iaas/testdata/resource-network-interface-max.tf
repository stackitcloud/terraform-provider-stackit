variable "parent_container_id" {}
variable "name" {}
variable "allowed_address" {}
variable "ipv4" {}
variable "ipv4_prefix" {}
variable "security" {}
variable "label" {}
variable "owner_email" {}
variable "network_name" {}
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
  project_id         = stackit_resourcemanager_project.example.project_id
  network_id         = stackit_network.network.network_id
  name               = var.network_name
  allowed_addresses  = var.security ? [var.allowed_address] : null
  ipv4               = var.ipv4
  security           = var.security
  security_group_ids = var.security ? [stackit_security_group.security_group.security_group_id] : null
  labels = {
    "acc-test" : var.label
  }
}

resource "stackit_public_ip" "public_ip" {
  project_id           = stackit_resourcemanager_project.example.project_id
  network_interface_id = stackit_network_interface.network_interface.network_interface_id
  labels = {
    "acc-test" : var.label
  }
}

resource "stackit_network_interface" "network_interface_simple" {
  project_id = stackit_resourcemanager_project.example.project_id
  network_id = stackit_network.network.network_id
}

resource "stackit_public_ip" "public_ip_simple" {
  project_id = stackit_resourcemanager_project.example.project_id
}

resource "stackit_public_ip_associate" "nic_public_ip_attach" {
  project_id           = stackit_resourcemanager_project.example.project_id
  network_interface_id = stackit_network_interface.network_interface_simple.network_interface_id
  public_ip_id         = stackit_public_ip.public_ip_simple.public_ip_id
}

resource "stackit_security_group" "security_group" {
  project_id = stackit_resourcemanager_project.example.project_id
  name       = var.network_name
}