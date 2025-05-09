variable "organization_id" {}

variable "name" {}
variable "transfer_network" {}
variable "network_ranges_prefix" {}

variable "route_prefix" {}
variable "route_next_hop" {}

resource "stackit_network_area" "network_area" {
  organization_id  = var.organization_id
  name             = var.name
  transfer_network = var.transfer_network
  network_ranges = [
    {
      prefix = var.network_ranges_prefix
    }
  ]
}

resource "stackit_network_area_route" "network_area_route" {
  organization_id = stackit_network_area.network_area.organization_id
  network_area_id = stackit_network_area.network_area.network_area_id
  prefix          = var.route_prefix
  next_hop        = var.route_next_hop
}