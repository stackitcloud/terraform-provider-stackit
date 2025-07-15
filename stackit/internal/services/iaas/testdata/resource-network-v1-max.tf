variable "project_id" {}
variable "name" {}
variable "ipv4_gateway" {}
variable "ipv4_nameserver_0" {}
variable "ipv4_nameserver_1" {}
variable "ipv4_prefix" {}
variable "ipv4_prefix_length" {}
variable "routed" {}
variable "label" {}

resource "stackit_network" "network_prefix" {
  project_id         = var.project_id
  name               = var.name
  ipv4_gateway       = var.ipv4_gateway != "" ? var.ipv4_gateway : null
  no_ipv4_gateway    = var.ipv4_gateway != "" ? null : true
  ipv4_nameservers   = [var.ipv4_nameserver_0, var.ipv4_nameserver_1]
  ipv4_prefix        = var.ipv4_prefix
  routed             = var.routed
  labels = {
    "acc-test" : var.label
  }
}


resource "stackit_network" "network_prefix_length" {
  project_id         = var.project_id
  name               = var.name
  no_ipv4_gateway    = true
  ipv4_nameservers   = [var.ipv4_nameserver_0, var.ipv4_nameserver_1]
  ipv4_prefix_length = var.ipv4_prefix_length
  routed             = var.routed
  labels = {
    "acc-test" : var.label
  }
}