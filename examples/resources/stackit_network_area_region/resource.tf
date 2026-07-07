resource "stackit_network_area_region" "example" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_area_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  ipv4 = {
    transfer_network = "10.1.2.0/24"
    network_ranges = [
      {
        prefix = "10.0.0.0/16"
      }
    ]
  }
}
