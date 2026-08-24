variable "organization_id" {}

variable "name" {}
variable "transfer_network" {}
variable "network_ranges_prefix" {}
variable "route_destination_type" {}
variable "route_destination_value" {}
variable "route_next_hop_type" {}
variable "route_next_hop_value" {}
variable "label" {}

resource "stackit_network_area" "network_area" {
  organization_id = var.organization_id
  name            = var.name
}

resource "stackit_network_area_region" "network_area_region" {
  organization_id = var.organization_id
  network_area_id = stackit_network_area.network_area.network_area_id
  ipv4 = {
    transfer_network = var.transfer_network
    network_ranges = [
      {
        prefix = var.network_ranges_prefix
      }
    ]
  }
}

resource "stackit_network_area_route" "network_area_route" {
  organization_id = var.organization_id
  network_area_id = stackit_network_area_region.network_area_region.network_area_id
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
