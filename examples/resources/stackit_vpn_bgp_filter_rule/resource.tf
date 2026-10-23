resource "stackit_vpn_bgp_filter_rule" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  gateway_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  filter_id  = stackit_vpn_bgp_filter.example.filter_id
  action     = "PERMIT"

  match = {
    prefixes          = ["10.0.0.0/16"]
    max_prefix_length = 24
  }

  set = {
    local_preference = 150
  }
}
