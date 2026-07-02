variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "plan_id" {}
variable "routing_type" {}
variable "az_tunnel1" {}
variable "az_tunnel2" {}
variable "local_asn" {}
variable "advertised_route_1" {}
variable "advertised_route_2" {}
variable "advertised_route_3" {
  default = ""
}
variable "label_key" {}
variable "label_value" {}

resource "stackit_vpn_gateway" "gateway" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
  plan_id      = var.plan_id
  routing_type = var.routing_type

  availability_zones = {
    tunnel1 = var.az_tunnel1
    tunnel2 = var.az_tunnel2
  }

  bgp = {
    local_asn                  = var.local_asn
    override_advertised_routes = compact([var.advertised_route_1, var.advertised_route_2, var.advertised_route_3])
  }

  labels = var.label_key == "" ? {} : {
    (var.label_key) = var.label_value
  }
}
