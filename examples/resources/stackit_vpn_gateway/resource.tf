resource "stackit_vpn_gateway" "example" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  display_name = "example-vpn-gateway"
  plan_id      = "p500"
  routing_type = "ROUTE_BASED"

  availability_zones = {
    tunnel1 = "eu01-1"
    tunnel2 = "eu01-2"
  }

  network_config = {
    predefined_network_prefix = ["10.20.0.0/28"]
  }
}
