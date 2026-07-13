resource "stackit_network_area" "example" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example-network-area"
  labels = {
    "key" = "value"
  }
}