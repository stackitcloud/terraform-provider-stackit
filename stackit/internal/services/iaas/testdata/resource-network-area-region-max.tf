variable "organization_id" {}

variable "name" {}
variable "transfer_network" {}
variable "network_ranges_prefix" {}
variable "default_prefix_length" {}
variable "min_prefix_length" {}
variable "max_prefix_length" {}
variable "default_nameservers" {}

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
    default_prefix_length = var.default_prefix_length
    min_prefix_length     = var.min_prefix_length
    max_prefix_length     = var.max_prefix_length
    default_nameservers = [
      var.default_nameservers
    ]
  }
}
