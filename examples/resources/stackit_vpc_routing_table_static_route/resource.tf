resource "stackit_vpc_routing_table_static_route" "example" {
  project_id       = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  vpc_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  routing_table_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  destination = {
    type  = "cidrv4"
    value = "192.168.0.0/24"
  }
  nexthop = {
    type  = "ipv4"
    value = "10.0.0.8"
  }
  labels = {
    "key" = "value"
  }
}
