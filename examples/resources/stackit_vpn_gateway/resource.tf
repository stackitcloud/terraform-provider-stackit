resource "stackit_vpn_gateway" "example" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "example-vpn-gateway"
  plan_id      = "p500"
  routing_type = "ROUTE_BASED"

  availability_zones = {
    tunnel1 = "eu01-1"
    tunnel2 = "eu01-2"
  }
}

# Only use the import statement, if you want to import an existing VPN gateway
import {
  to = stackit_vpn_gateway.example
  id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx,eu01,xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
