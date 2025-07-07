resource "stackit_routing_table_route" "example" {
  organization_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_area_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  routing_table_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  destination = {
    type  = "cidrv4"
    value = "192.168.178.0/24"
  }
  next_hop = {
    type  = "ipv4"
    value = "192.168.178.1"
  }
  labels = {
    "key" = "value"
  }
}