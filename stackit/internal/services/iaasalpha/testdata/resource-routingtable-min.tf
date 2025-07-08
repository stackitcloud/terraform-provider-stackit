variable "organization_id" {}
variable "network_area_id" {}
variable "name" {}

resource "stackit_routing_table" "routing_table" {
  organization_id = var.organization_id
  network_area_id = var.network_area_id
  name            = var.name
}
