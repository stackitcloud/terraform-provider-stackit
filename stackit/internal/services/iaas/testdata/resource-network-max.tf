variable "project_id" {}
variable "name" {}
variable "ipv4_gateway" {}
variable "ipv4_nameservers" {}
variable "ipv4_prefix" {}
variable "ipv4_prefix_length" {}
variable "routed" {}
variable "label" {}

resource "stackit_network" "network" {
  project_id         = var.project_id
  name               = var.name
  ipv4_gateway       = var.ipv4_gateway != "" ? var.ipv4_gateway : null
  no_ipv4_gateway    = var.ipv4_gateway != "" ? null : true
  ipv4_nameservers   = [var.ipv4_nameservers]
  ipv4_prefix        = var.ipv4_prefix
  ipv4_prefix_length = var.ipv4_prefix_length
  routed             = var.routed
  labels = {
    "acc-test" : var.label
  }
}