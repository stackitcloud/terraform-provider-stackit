variable "organization_id" {}
variable "network_area_id" {}
variable "routing_table_name" {}
variable "destination_type" {}
variable "destination_value" {}
variable "next_hop_type" {}
variable "next_hop_value" {}

resource "stackit_routing_table" "routing_table" {
  organization_id = var.organization_id
  network_area_id = var.network_area_id
  name            = var.routing_table_name
}

resource "stackit_routing_table_route" "route" {
  organization_id  = var.organization_id
  network_area_id  = var.network_area_id
  routing_table_id = stackit_routing_table.routing_table.routing_table_id
  destination = {
    type  = var.destination_type
    value = var.destination_value
  }
  next_hop = {
    type  = var.next_hop_type
    value = var.next_hop_value
  }
}
