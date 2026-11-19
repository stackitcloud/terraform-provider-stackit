variable "rule_action" {}
variable "rule_peer" {}
variable "rule_prefix" {}
variable "rule_local_preference" {}

resource "stackit_vpn_bgp_filter_rule" "rule" {
  project_id = stackit_vpn_bgp_filter.filter.project_id
  region     = stackit_vpn_bgp_filter.filter.region
  gateway_id = stackit_vpn_bgp_filter.filter.gateway_id
  filter_id  = stackit_vpn_bgp_filter.filter.filter_id
  action     = var.rule_action

  match = {
    peer     = var.rule_peer
    prefixes = [var.rule_prefix]
  }

  set = {
    local_preference = var.rule_local_preference
  }
}
