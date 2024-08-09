resource "stackit_network_area" "example" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example-network-area"
  network_ranges = [
    {
      prefix = "1.2.3.4"
    }
  ]
  transfer_network = "1.2.3.4/5"
}
