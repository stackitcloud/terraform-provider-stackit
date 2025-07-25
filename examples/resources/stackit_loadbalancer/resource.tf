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
  machine_type      = "g1.1"
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

# This example shows an advanced setup where the load balancer is in one
# network and the target server is in another. This requires manual
# security group configuration.

# 1. Create a network for the Load Balancer
resource "stackit_network" "lb_network" {
  project_id  = var.project_id
  name        = "lb-network"
  ipv4_prefix = "192.168.1.0/24"
}

# 2. Create a separate network for the Target Server
resource "stackit_network" "target_network" {
  project_id  = var.project_id
  name        = "target-network"
  ipv4_prefix = "192.168.2.0/24"
}

# 3. Create the Load Balancer and disable automatic security groups
resource "stackit_loadbalancer" "example_advanced" {
  project_id = var.project_id
  name       = "advanced-lb"

  # This is the key setting for manual mode
  disable_security_group_assignment = true

  networks = [{
    network_id = stackit_network.lb_network.id
    role       = "ROLE_LISTENERS_AND_TARGETS"
  }]

  target_pools = [{
    name        = "cross-network-pool"
    target_port = 80
    targets = [{
      display_name = "remote-target-server"
      ip           = stackit_server.target_server.network_interfaces[0].ipv4
    }]
  }]

  listeners = [{
    port        = 80
    protocol    = "PROTOCOL_TCP"
    target_pool = "cross-network-pool"
  }]
}

# 4. Create a new security group for the target server
resource "stackit_security_group" "target_sg" {
  project_id  = var.project_id
  name        = "target-sg-for-lb-access"
  description = "Allows ingress traffic from the advanced load balancer."
}

# 5. Create a rule to allow traffic FROM the load balancer
#    This is the core of the manual setup.
resource "stackit_security_group_rule" "allow_lb_ingress" {
  security_group_id = stackit_security_group.target_sg.id
  direction         = "ingress"
  protocol          = "tcp"

  # Use the computed security_group_id from the load balancer
  remote_security_group_id = stackit_loadbalancer.example_advanced.security_group_id

  port_range = {
    min = 80
    max = 80
  }
}

# 6. Create the target server and assign the new security group to it
resource "stackit_server" "target_server" {
  project_id        = var.project_id
  name              = "remote-target-server"
  machine_type      = "c1.1"
  availability_zone = "eu01-1"

  boot_volume = {
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" // e.g., an Ubuntu image ID
    size        = 10
  }

  network_interfaces = [{
    network_id      = stackit_network.target_network.id
    security_groups = [stackit_security_group.target_sg.id]
  }]

  # Ensure the rule is created before the server
  depends_on = [stackit_security_group_rule.allow_lb_ingress]
}

# Only use the import statement, if you want to import an existing loadbalancer
import {
  to = stackit_loadbalancer.import-example
  id = "${var.project_id},${var.region},${var.loadbalancer_name}"
}