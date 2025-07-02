resource "stackit_routing_table" "example" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_area_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example"
  labels = {
    "key" = "value"
  }
}
