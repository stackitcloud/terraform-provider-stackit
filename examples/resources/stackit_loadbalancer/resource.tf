variable "project_id" {
  description = "The STACKIT Project ID"
  type        = string
  default     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

variable "image_id" {
  description = "A valid Debian 12 Image ID available in all projects"
  type        = string
  default     = "939249d1-6f48-4ab7-929b-95170728311a"
}

# Create a network
resource "stackit_network" "example_network" {
  project_id       = var.project_id
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
  project_id = var.project_id
  network_id = stackit_network.example_network.network_id
}

# Create a public IP for the load balancer
resource "stackit_public_ip" "public-ip" {
  project_id = var.project_id
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
  project_id = var.project_id
  name       = "example-server"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "7b10e105-295b-4369-b6e0-567ec940a02b" // Ubuntu 24.04
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
  project_id = var.project_id
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
# and `load_balancer_security_group_id` attributes.

# We create two separate networks: one for the load balancer and one for the target.
resource "stackit_network" "lb_network" {
  project_id       = var.project_id
  name             = "lb-network-example"
  ipv4_prefix      = "192.168.10.0/25"
  ipv4_nameservers = ["8.8.8.8"]
  routed           = true
}

resource "stackit_network" "target_network" {
  project_id       = var.project_id
  name             = "target-network-example"
  ipv4_prefix      = "192.168.15.0/25"
  ipv4_nameservers = ["8.8.8.8"]
}

resource "stackit_public_ip" "example" {
  project_id = var.project_id
}

resource "stackit_loadbalancer" "example" {
  project_id       = var.project_id
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
  project_id  = var.project_id
  name        = "target-sg-for-lb-access"
  description = "Allows ingress traffic from the example load balancer."
}

# Create a rule to allow traffic FROM the load balancer.
# This rule uses the computed `load_balancer_security_group_id` of the load balancer.
resource "stackit_security_group_rule" "allow_lb_ingress" {
  project_id        = var.project_id
  security_group_id = stackit_security_group.target_sg.security_group_id
  direction         = "ingress"
  protocol = {
    name = "tcp"
  }

  # This is the crucial link: it allows traffic from the LB's security group.
  remote_security_group_id = stackit_loadbalancer.example.load_balancer_security_group_id

  port_range = {
    min = 80
    max = 80
  }
}

resource "stackit_server" "example" {
  project_id        = var.project_id
  name              = "example-remote-target"
  machine_type      = "g2i.2"
  availability_zone = "eu01-1"

  boot_volume = {
    source_type = "image"
    source_id   = var.image_id
    size        = 10
  }

  network_interfaces = [
    stackit_network_interface.nic.network_interface_id
  ]
}

resource "stackit_network_interface" "nic" {
  project_id         = var.project_id
  network_id         = stackit_network.target_network.network_id
  security_group_ids = [stackit_security_group.target_sg.security_group_id]
}
# End of advanced example

# Only use the import statement, if you want to import an existing loadbalancer
import {
  to = stackit_loadbalancer.import-example
  id = "${var.project_id},${var.region},${var.loadbalancer_name}"
}
