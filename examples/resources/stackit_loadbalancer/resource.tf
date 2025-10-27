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

# Create a public IP for the load balancer
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
  machine_type      = "g2i.1"
  keypair_name      = stackit_key_pair.keypair.name
  network_interfaces = [
    stackit_network_interface.nic.network_interface_id
  ]
}

# Create a load balancer
resource "stackit_loadbalancer" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-load-balancer"
  plan_id    = "p10"
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
      tcp = {
        idle_timeout = "90s"
      }
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

# This example demonstrates an advanced setup where the Load Balancer is in one
# network and the target server is in another. This requires manual
# security group configuration using the `disable_security_group_assignment`
# and `security_group_id` attributes.

# We create two separate networks: one for the load balancer and one for the target.
resource "stackit_network" "lb_network" {
  project_id       = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name             = "lb-network-example"
  ipv4_prefix      = "192.168.10.0/25"
  ipv4_nameservers = ["8.8.8.8"]
}

resource "stackit_network" "target_network" {
  project_id       = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name             = "target-network-example"
  ipv4_prefix      = "192.168.10.0/25"
  ipv4_nameservers = ["8.8.8.8"]
}

resource "stackit_public_ip" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "stackit_loadbalancer" "example" {
  project_id       = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name             = "example-advanced-lb"
  external_address = stackit_public_ip.example.ip

  # Key setting for manual mode: disables automatic security group handling.
  disable_security_group_assignment = true

  networks = [{
    network_id = stackit_network.lb_network.network_id
    role       = "ROLE_LISTENERS_AND_TARGETS"
  }]

  listeners = [{
    port        = 80
    protocol    = "PROTOCOL_TCP"
    target_pool = "cross-network-pool"
  }]

  target_pools = [{
    name        = "cross-network-pool"
    target_port = 80
    targets = [{
      display_name = stackit_server.example.name
      ip           = stackit_network_interface.nic.ipv4
    }]
  }]
}

# Create a new security group to be assigned to the target server.
resource "stackit_security_group" "target_sg" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "target-sg-for-lb-access"
  description = "Allows ingress traffic from the example load balancer."
}

# Create a rule to allow traffic FROM the load balancer.
# This rule uses the computed `security_group_id` of the load balancer.
resource "stackit_security_group_rule" "allow_lb_ingress" {
  project_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  security_group_id = stackit_security_group.target_sg.security_group_id
  direction         = "ingress"
  protocol = {
    name = "tcp"
  }

  # This is the crucial link: it allows traffic from the LB's security group.
  remote_security_group_id = stackit_loadbalancer.example.security_group_id

  port_range = {
    min = 80
    max = 80
  }
}

resource "stackit_server" "example" {
  project_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name              = "example-remote-target"
  machine_type      = "g2i.2"
  availability_zone = "eu01-1"

  boot_volume = {
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    size        = 10
  }

  network_interfaces = [
    stackit_network_interface.nic.network_interface_id
  ]
}

resource "stackit_network_interface" "nic" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_id         = stackit_network.target_network.network_id
  security_group_ids = [stackit_security_group.target_sg.security_group_id]
}
# End of advanced example

# Only use the import statement, if you want to import an existing loadbalancer
import {
  to = stackit_loadbalancer.import-example
  id = "${var.project_id},${var.region},${var.loadbalancer_name}"
}
