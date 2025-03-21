# Create a network
resource "stackit_network" "example_network" {
  project_id       = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name             = "example-network"
  ipv4_nameservers = ["8.8.8.8"]
  ipv4_prefix      = "192.168.0.0/25"
  labels = {
    "key" = "value"
  }
  routed = true
}

# Create a network interface
resource "stackit_network_interface" "nic" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_id = stackit_network.example_network.network_id
}

# Create a public IP for the load balance
resource "stackit_public_ip" "public-ip" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  lifecycle {
    ignore_changes = [network_interface_id]
  }
}

# Create a key pair for accessing the server instance
resource "stackit_key_pair" "keypair" {
  name       = "example-key-pair"
  public_key = chomp(file("path/to/id_rsa.pub"))
}

# Create a server instance
resource "stackit_server" "boot-from-image" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-server"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "59838a89-51b1-4892-b57f-b3caf598ee2f" // Ubuntu 24.04
  }
  availability_zone = "xxxx-x"
  machine_type      = "g1.1"
  keypair_name      = stackit_key_pair.keypair.name
}

# Attach the network interface to the server
resource "stackit_server_network_interface_attach" "nic-attachment" {
  project_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id            = stackit_server.boot-from-image.server_id
  network_interface_id = stackit_network_interface.nic.network_interface_id
}

# Create a load balancer
resource "stackit_loadbalancer" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-load-balancer"
  target_pools = [
    {
      name        = "example-target-pool"
      target_port = 80
      targets = [
        {
          display_name = stackit_server.boot-from-image.name
          ip           = stackit_network_interface.nic.ipv4
        }
      ]
      active_health_check = {
        healthy_threshold   = 10
        interval            = "3s"
        interval_jitter     = "3s"
        timeout             = "3s"
        unhealthy_threshold = 10
      }
    }
  ]
  listeners = [
    {
      display_name = "example-listener"
      port         = 80
      protocol     = "PROTOCOL_TCP"
      target_pool  = "example-target-pool"
    }
  ]
  networks = [
    {
      network_id = stackit_network.example_network.network_id
      role       = "ROLE_LISTENERS_AND_TARGETS"
    }
  ]
  external_address = stackit_public_ip.public-ip.ip
  options = {
    private_network_only = false
  }
}