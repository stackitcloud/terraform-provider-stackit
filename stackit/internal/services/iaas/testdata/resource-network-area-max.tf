variable "organization_id" {}

variable "name" {}
variable "transfer_network" {}
variable "network_ranges_prefix" {}
variable "default_nameservers" {}
variable "default_prefix_length" {}
variable "max_prefix_length" {}
variable "min_prefix_length" {}

variable "route_destination_type" {}
variable "route_destination_value" {}
variable "route_next_hop_type" {}
variable "route_next_hop_value" {}
variable "label" {}

resource "stackit_network_area" "network_area" {
  organization_id = var.organization_id
  name            = var.name
  network_ranges = [
    {
      prefix = var.network_ranges_prefix
    }
  ]
  transfer_network      = var.transfer_network
  default_nameservers   = [var.default_nameservers]
  default_prefix_length = var.default_prefix_length
  max_prefix_length     = var.max_prefix_length
  min_prefix_length     = var.min_prefix_length
  labels = {
    "acc-test" : var.label
  }
}

resource "stackit_network_area_route" "network_area_route" {
  organization_id = stackit_network_area.network_area.organization_id
  network_area_id = stackit_network_area.network_area.network_area_id
  destination = {
    type  = var.route_destination_type
    value = var.route_destination_value
  }
  next_hop = {
    type  = var.route_next_hop_type
    value = var.route_next_hop_value
  }
  labels = {
    "acc-test" : var.label
  }
}