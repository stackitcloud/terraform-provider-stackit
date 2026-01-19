variable "organization_id" {}
variable "network_area_id" {}
variable "name" {}
variable "description" {}
variable "region" {}
variable "label" {}
variable "system_routes" {}
variable "dynamic_routes" {}

resource "stackit_routing_table" "routing_table" {
  organization_id = var.organization_id
  network_area_id = var.network_area_id
  name            = var.name
  description     = var.description
  region          = var.region
  labels = {
    "acc-test" : var.label
  }
  system_routes  = var.system_routes
  dynamic_routes = var.dynamic_routes
}
