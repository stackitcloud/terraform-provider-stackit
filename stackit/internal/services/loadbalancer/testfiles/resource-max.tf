
variable "project_id" {}
variable "network_name" {}
variable "server_name" {}

variable "loadbalancer_name" {}
variable "plan_id" {}
variable "target_pool_name" {}
variable "target_port" {}
variable "target_display_name" {}
variable "listener_port" {}
variable "listener_protocol" {}
variable "network_role" {}
variable "disable_security_group_assignment" {}

variable "listener_display_name" {}
variable "listener_server_name_indicators" {}
variable "healthy_threshold" {}
variable "health_interval" {}
variable "health_interval_jitter" {}
variable "health_timeout" {}
variable "unhealthy_threshold" {}
variable "use_source_ip_address" {}
variable "private_network_only" {}
variable "acl" {}

variable "observability_logs_push_url" {}
variable "observability_metrics_push_url" {}
variable "observability_credential_logs_name" {}
variable "observability_credential_logs_username" {}
variable "observability_credential_logs_password" {}
variable "observability_credential_metrics_name" {}
variable "observability_credential_metrics_username" {}
variable "observability_credential_metrics_password" {}

resource "stackit_network" "network" {
  project_id       = var.project_id
  name             = var.network_name
  ipv4_nameservers = ["8.8.8.8"]
  ipv4_prefix      = "192.168.3.0/25"
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
  plan_id    = var.plan_id
  disable_security_group_assignment = var.disable_security_group_assignment
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
      active_health_check = {
        healthy_threshold   = var.healthy_threshold
        interval            = var.health_interval
        interval_jitter     = var.health_interval_jitter
        timeout             = var.health_timeout
        unhealthy_threshold = var.unhealthy_threshold
      }
      session_persistence = {
        use_source_ip_address = var.use_source_ip_address
      }
    }
  ]
  listeners = [
    {
      display_name = var.listener_display_name
      port         = var.listener_port
      protocol     = var.listener_protocol
      target_pool  = var.target_pool_name
      server_name_indicators = [
        {
          name = var.listener_server_name_indicators
        }
      ]
    }
  ]
  networks = [
    {
      network_id = stackit_network.network.network_id
      role       = var.network_role
    }
  ]
  options = {
    private_network_only = var.private_network_only
    acl                  = [var.acl]
    observability = {
    	logs = {
    		credentials_ref = stackit_loadbalancer_observability_credential.logs.credentials_ref
    		push_url = var.observability_logs_push_url
    	}
    	metrics = {
    		credentials_ref = stackit_loadbalancer_observability_credential.metrics.credentials_ref
    		push_url = var.observability_metrics_push_url
    	}
    }
  }
  external_address = stackit_public_ip.public_ip.ip
}

resource "stackit_loadbalancer_observability_credential" "logs" {
  project_id   = var.project_id
  display_name = var.observability_credential_logs_name
  username     = var.observability_credential_logs_username
  password     = var.observability_credential_logs_password
}

resource "stackit_loadbalancer_observability_credential" "metrics" {
  project_id   = var.project_id
  display_name = var.observability_credential_metrics_name
  username     = var.observability_credential_metrics_username
  password     = var.observability_credential_metrics_password
}

