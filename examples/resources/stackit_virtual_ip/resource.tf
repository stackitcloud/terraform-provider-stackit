// Create a network
resource "stackit_network" "network" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-network"
}

// Create a virtual IP on the network
resource "stackit_virtual_ip" "virtual_ip" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_id = stackit_network.network.network_id
  name       = "example-virtual-ip"
  labels = {
    "key" = "value"
  }
}
