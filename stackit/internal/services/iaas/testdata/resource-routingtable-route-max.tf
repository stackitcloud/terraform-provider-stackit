variable "organization_id" {}
variable "name" {}
variable "routing_table_name" {}
variable "destination_type" {}
variable "destination_value" {}
variable "next_hop_type" {}
variable "next_hop_value" {}
variable "label" {}

resource "stackit_network_area" "network_area" {
  organization_id = var.organization_id
  name            = var.name
  labels = {
    "preview/routingtables" = "true"
  }
}

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

resource "stackit_routing_table" "routing_table" {
  organization_id = var.organization_id
  network_area_id = stackit_network_area.network_area.network_area_id
  name            = var.routing_table_name

  depends_on = [stackit_network_area_region.network_area_region]
}

resource "stackit_routing_table_route" "route" {
  organization_id  = var.organization_id
  network_area_id = stackit_network_area.network_area.network_area_id
  routing_table_id = stackit_routing_table.routing_table.routing_table_id
  destination = {
    type  = var.destination_type
    value = var.destination_value
  }
  next_hop = {
    type  = var.next_hop_type
    value = var.next_hop_value
  }
  labels = {
    "acc-test" = var.label
  }
}
