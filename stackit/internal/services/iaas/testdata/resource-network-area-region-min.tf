variable "organization_id" {}

variable "name" {}
variable "transfer_network" {}
variable "network_ranges_prefix" {}

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
