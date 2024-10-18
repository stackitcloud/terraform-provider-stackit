resource "stackit_network_area_route" "example" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_area_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  prefix          = "192.168.0.0/24"
  next_hop        = "192.168.0.0"
  labels = {
    "key" = "value"
  }
}
