
variable "project_id" {}
variable "network_name" {}
variable "server_name" {}

variable "loadbalancer_name" {}
variable "target_pool_name" {}
variable "target_port" {}
variable "target_display_name" {}
variable "listener_port" {}
variable "listener_protocol" {}
variable "network_role" {}

variable "obs_display_name" {}
variable "obs_username" {}
variable "obs_password" {}

resource "stackit_network" "network" {
  project_id       = var.project_id
  name             = var.network_name
  ipv4_nameservers = ["8.8.8.8"]
  ipv4_prefix      = "192.168.2.0/25"
  routed           = "true"
}

resource "stackit_network_interface" "network_interface" {
  project_id = stackit_network.network.project_id
  network_id = stackit_network.network.network_id
  name       = "name"
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
  availability_zone = "eu01-1"
  name              = var.server_name
  machine_type      = "t1.1"
  boot_volume = {
    size                  = 32
    source_type           = "image"
    source_id             = "59838a89-51b1-4892-b57f-b3caf598ee2f"
    delete_on_termination = "true"
  }
  network_interfaces = [stackit_network_interface.network_interface.network_interface_id]
  user_data          = "#!/bin/bash"
}

resource "stackit_loadbalancer" "loadbalancer" {
  project_id = var.project_id
  name       = var.loadbalancer_name
  target_pools = [
    {
      name        = var.target_pool_name
      target_port = var.target_port
      targets = [
        {
          display_name = var.target_display_name
          ip           = stackit_network_interface.network_interface.ipv4
        }
      ]
    }
  ]
  listeners = [
    {
      port        = var.listener_port
      protocol    = var.listener_protocol
      target_pool = var.target_pool_name
    }
  ]
  networks = [
    {
      network_id = stackit_network.network.network_id
      role       = var.network_role
    }
  ]
  external_address = stackit_public_ip.public_ip.ip
}

resource "stackit_loadbalancer_observability_credential" "obs_credential" {
  project_id   = var.project_id
  display_name = var.obs_display_name
  username     = var.obs_username
  password     = var.obs_password
}
