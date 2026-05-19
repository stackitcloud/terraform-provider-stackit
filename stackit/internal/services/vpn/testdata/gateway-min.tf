variable "project_id" {}
variable "display_name" {}
variable "plan_id" {}
variable "routing_type" {}
variable "az_tunnel1" {}
variable "az_tunnel2" {}

resource "stackit_vpn_gateway" "gateway" {
  project_id   = var.project_id
  display_name = var.display_name
  plan_id      = var.plan_id
  routing_type = var.routing_type

  availability_zones = {
    tunnel1 = var.az_tunnel1
    tunnel2 = var.az_tunnel2
  }
}
