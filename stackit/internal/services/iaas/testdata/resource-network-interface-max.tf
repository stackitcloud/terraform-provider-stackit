variable "project_id" {}
variable "name" {}
variable "allowed_address" {}
variable "ipv4" {}
variable "ipv4_prefix" {}
variable "security" {}
variable "label" {}

resource "stackit_network" "network" {
  project_id  = var.project_id
  name        = var.name
  ipv4_prefix = var.ipv4_prefix
}

resource "stackit_network_interface" "network_interface" {
  project_id         = var.project_id
  network_id         = stackit_network.network.network_id
  name               = var.name
  allowed_addresses  = var.security ? [var.allowed_address] : null
  ipv4               = var.ipv4
  security           = var.security
  security_group_ids = var.security ? [stackit_security_group.security_group.security_group_id] : null
  labels = {
    "acc-test" : var.label
  }
}

resource "stackit_security_group" "security_group" {
  project_id = var.project_id
  name       = var.name
}