// backend server data
variable "image_id" {
  description = "A valid Image ID available in the project for the target server"
  type        = string
  default     = "939249d1-6f48-4ab7-929b-95170728311a"
}
variable "availability_zone" {
  description = "The availability zone"
  type        = string
  default     = "eu01-1"
}
variable "machine_type" {
  description = "The machine flavor"
  type        = string
  default     = "c2i.1"
}
variable "server_name_min" {
  description = "The name of the backend server"
  type        = string
  default     = "backend-server-min"
}

// general data
variable "project_id" {}
variable "region" {}

// load balancer data
variable "loadbalancer_name" {}
variable "network_role" {}
variable "network_name" {}
variable "plan_id" {}
variable "listener_name" {}
variable "listener_port" {}
variable "host" {}
variable "protocol_http" {}
variable "target_pool_name" {}
variable "target_pool_port" {}

resource "stackit_network" "network" {
  project_id       = var.project_id
  name             = var.network_name
  ipv4_nameservers = ["1.1.1.1"]
  ipv4_prefix      = "192.168.3.0/25"
  routed           = "true"
}

resource "stackit_network_interface" "network_interface" {
  project_id = var.project_id
  network_id = stackit_network.network.network_id
  lifecycle {
    ignore_changes = [
      security_group_ids,
    ]
  }
}

resource "stackit_public_ip" "public_ip" {
  project_id           = var.project_id
  network_interface_id = stackit_network_interface.network_interface.network_interface_id
  lifecycle {
    ignore_changes = [
      network_interface_id
    ]
  }
}

resource "stackit_server" "server" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  name              = var.server_name_min
  machine_type      = var.machine_type
  boot_volume = {
    size                  = 20
    source_type           = "image"
    source_id             = var.image_id
    delete_on_termination = "true"
  }
  network_interfaces = [
    stackit_network_interface.network_interface.network_interface_id
  ]
  # Explicit dependencies to ensure ordering
  depends_on = [
    stackit_network.network,
    stackit_network_interface.network_interface
  ]
}

resource "stackit_application_load_balancer" "loadbalancer" {
  region     = var.region
  project_id = var.project_id
  name       = var.loadbalancer_name
  plan_id    = var.plan_id
  target_pools = [
    {
      name        = var.target_pool_name
      target_port = var.target_pool_port
      targets = [
        {
          ip = stackit_network_interface.network_interface.ipv4
        }
      ]
    }
  ]
  listeners = [{
    name = var.listener_name
    port = var.listener_port
    http = {
      hosts = [{
        host = var.host
        rules = [{
          target_pool = var.target_pool_name
        }]
      }]
    }
    protocol = var.protocol_http
  }]
  networks = [
    {
      network_id = stackit_network.network.network_id
      role       = var.network_role
    }
  ]
  external_address = stackit_public_ip.public_ip.ip
}
