# Create a network
resource "openstack_networking_network_v2" "example" {
  name = "example-network"
}

# Create a subnet
resource "openstack_networking_subnet_v2" "example" {
  name            = "example-subnet"
  cidr            = "192.168.0.0/25"
  dns_nameservers = ["8.8.8.8"]
  network_id      = openstack_networking_network_v2.example.id
}

# Get public network
data "openstack_networking_network_v2" "public" {
  name = "floating-net"
}

# Create a floating IP
resource "openstack_networking_floatingip_v2" "example" {
  pool = data.openstack_networking_network_v2.public.name
}

# Get flavor for instance
data "openstack_compute_flavor_v2" "example" {
  name = "g1.1"
}

# Create an instance
resource "openstack_compute_instance_v2" "example" {
  depends_on      = [openstack_networking_subnet_v2.example]
  name            = "example-instance"
  flavor_id       = data.openstack_compute_flavor_v2.example.id
  admin_pass      = "example"
  security_groups = ["default"]

  block_device {
    uuid                  = "4364cdb2-dacd-429b-803e-f0f7cfde1c24" // Ubuntu 22.04
    volume_size           = 32
    source_type           = "image"
    destination_type      = "volume"
    delete_on_termination = true
  }

  network {
    name = openstack_networking_network_v2.example.name
  }

  lifecycle {
    # Security groups are modified by the STACKIT LoadBalancer Service, so terraform should ignore changes here
    ignore_changes = [security_groups]
  }
}

# Create a router and attach it to the public network
resource "openstack_networking_router_v2" "example" {
  name                = "example-router"
  admin_state_up      = "true"
  external_network_id = data.openstack_networking_network_v2.public.id
}

# Attach the subnet to the router
resource "openstack_networking_router_interface_v2" "example_interface" {
  router_id = openstack_networking_router_v2.example.id
  subnet_id = openstack_networking_subnet_v2.example.id
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
          display_name = "example-target"
          ip           = openstack_compute_instance_v2.example.network.0.fixed_ip_v4
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
      network_id = openstack_networking_network_v2.example.id
      role       = "ROLE_LISTENERS_AND_TARGETS"
    }
  ]
  external_address = openstack_networking_floatingip_v2.example.address
  options = {
    private_network_only = false
  }
}