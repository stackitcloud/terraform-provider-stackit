// Create a network
resource "stackit_network" "network" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-network"
}

// Create a network interface
resource "stackit_network_interface" "network_interface" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_id = stackit_network.network.network_id
  name       = "example-network-interface"
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

// Add a network interface as a member of the virtual IP
resource "stackit_virtual_ip_member" "virtual_ip_member" {
  project_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_id           = stackit_network.network.network_id
  virtual_ip_id        = stackit_virtual_ip.vip.virtual_ip_id
  network_interface_id = stackit_network_interface.nic_1.network_interface_id
}
