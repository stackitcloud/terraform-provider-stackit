variable "filter_display_name" {}

resource "stackit_vpn_bgp_filter" "filter" {
  project_id   = stackit_vpn_gateway.gateway.project_id
  region       = stackit_vpn_gateway.gateway.region
  gateway_id   = stackit_vpn_gateway.gateway.gateway_id
  display_name = var.filter_display_name
}
