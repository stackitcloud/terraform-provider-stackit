resource "stackit_network_area" "example" {
  organization_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name               = "example-network-area"
  network_ranges     = ["1.2.3.4", "5.6.7.8"]
  transfer_network   = "1.2.3.4/5"
}
