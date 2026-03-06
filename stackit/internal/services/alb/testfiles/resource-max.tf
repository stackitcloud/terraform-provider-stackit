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
variable "server_name_max" {
  description = "The name of the backend server"
  type        = string
  default     = "backend-server-max"
}

// observability
variable "observability_credential_name" {}
variable "observability_credential_username" {}
variable "observability_credential_password" {}

// general data
variable "project_id" {}
variable "region" {}

// load balancer data
variable "loadbalancer_name" {}
variable "plan_id" {}
variable "target_display_name" {}
variable "labels_key_1" {}
variable "labels_value_1" {}
variable "labels_key_2" {}
variable "labels_value_2" {}
variable "ahc_interval" {}
variable "ahc_interval_jitter" {}
variable "ahc_timeout" {}
variable "ahc_healthy_threshold" {}
variable "ahc_unhealthy_threshold" {}
variable "ahc_http_ok_status_200" {}
variable "ahc_http_ok_status_201" {}
variable "ahc_http_path" {}
variable "tls_config_enabled" {}
variable "tls_config_skip" {}
variable "tls_config_custom_ca" {}
variable "listener_port_1" {}
variable "host_1" {}
variable "target_pool_name_1" {}
variable "target_pool_port_1" {}
variable "web_socket" {}
variable "query_parameters_name_1" {}
variable "query_parameters_exact_match_1" {}
variable "query_parameters_name_2" {}
variable "query_parameters_exact_match_2" {}
variable "headers_name_1" {}
variable "headers_exact_match_1" {}
variable "headers_name_2" {}
variable "headers_exact_match_2" {}
variable "headers_name_3" {}
variable "path_prefix_1" {}
variable "path_prefix_2" {}
variable "target_pool_name_2" {}
variable "target_pool_port_2" {}
variable "host_3" {}
variable "path_prefix_3" {}
variable "target_pool_name_3" {}
variable "target_pool_port_3" {}
variable "listener_port_4" {}
variable "host_4" {}
variable "path_prefix_4" {}
variable "listener_name_1" {}
variable "listener_name_2" {}
variable "target_pool_name_4" {}
variable "target_pool_port_4" {}
variable "network_name_listener" {}
variable "network_name_targets" {}
variable "network_role_listeners" {}
variable "network_role_targets" {}
variable "disable_security_group_assignment" {}
variable "protocol_http" {}
variable "private_network_only" {}
variable "acl" {}
variable "ephemeral_address" {}
variable "observability_logs_push_url" {}
variable "observability_metrics_push_url" {}

resource "stackit_network" "listener_network" {
  project_id       = var.project_id
  name             = var.network_name_listener
  ipv4_nameservers = ["1.1.1.1"]
  ipv4_prefix      = "10.11.10.0/24"
  routed           = "true"
}

resource "stackit_network" "target_network" {
  project_id       = var.project_id
  name             = var.network_name_targets
  ipv4_nameservers = ["1.1.1.1"]
  ipv4_prefix      = "10.11.11.0/24"
  routed           = "true"
}

resource "stackit_network_interface" "network_interface_listener" {
  project_id = var.project_id
  network_id = stackit_network.listener_network.network_id
  lifecycle {
    ignore_changes = [
      security_group_ids,
    ]
  }
}

resource "stackit_network_interface" "network_interface_target" {
  project_id = var.project_id
  network_id = stackit_network.target_network.network_id
  lifecycle {
    ignore_changes = [
      security_group_ids,
    ]
  }
}

resource "stackit_server" "server_max" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  name              = var.server_name_max
  machine_type      = var.machine_type
  boot_volume = {
    size                  = 20
    source_type           = "image"
    source_id             = var.image_id
    delete_on_termination = "true"
  }
  network_interfaces = [
    stackit_network_interface.network_interface_target.network_interface_id
  ]
  # Explicit dependencies to ensure ordering
  depends_on = [
    stackit_network.target_network,
    stackit_network_interface.network_interface_target
  ]
}

resource "stackit_loadbalancer_observability_credential" "observer" {
  project_id   = var.project_id
  display_name = var.observability_credential_name
  password     = var.observability_credential_password
  username     = var.observability_credential_username
}

resource "stackit_application_load_balancer" "loadbalancer" {
  region                                   = var.region
  project_id                               = var.project_id
  name                                     = var.loadbalancer_name
  plan_id                                  = var.plan_id
  disable_target_security_group_assignment = var.disable_security_group_assignment
  labels = {
    (var.labels_key_1) = var.labels_value_1
    (var.labels_key_2) = var.labels_value_2
  }
  target_pools = [
    {
      name = var.target_pool_name_1
      active_health_check = {
        interval            = var.ahc_interval
        interval_jitter     = var.ahc_interval_jitter
        timeout             = var.ahc_timeout
        healthy_threshold   = var.ahc_healthy_threshold
        unhealthy_threshold = var.ahc_unhealthy_threshold
        http_health_checks = {
          ok_status = [var.ahc_http_ok_status_200, var.ahc_http_ok_status_201]
          path      = var.ahc_http_path
        }
      }
      target_port = var.target_pool_port_1
      targets = [
        {
          display_name = var.target_display_name
          ip           = stackit_network_interface.network_interface_target.ipv4
        }
      ]
      tls_config = {
        enabled                     = var.tls_config_enabled
        skip_certificate_validation = var.tls_config_skip
        custom_ca                   = var.tls_config_custom_ca
      }
      }, {
      name        = var.target_pool_name_2
      target_port = var.target_pool_port_2
      targets = [
        {
          display_name = var.target_display_name
          ip           = stackit_network_interface.network_interface_target.ipv4
        }
      ]
      }, {
      name        = var.target_pool_name_3
      target_port = var.target_pool_port_3
      targets = [
        {
          display_name = var.target_display_name
          ip           = stackit_network_interface.network_interface_target.ipv4
        }
      ]
      }, {
      name        = var.target_pool_name_4
      target_port = var.target_pool_port_4
      targets = [
        {
          display_name = var.target_display_name
          ip           = stackit_network_interface.network_interface_target.ipv4
        }
      ]
    }
  ]
  listeners = [{
    name = var.listener_name_1
    port = var.listener_port_1
    http = {
      hosts = [{
        host = var.host_1
        rules = [{
          target_pool = var.target_pool_name_1
          web_socket  = var.web_socket
          query_parameters = [{
            name        = var.query_parameters_name_1
            exact_match = var.query_parameters_exact_match_1
            }, {
            name        = var.query_parameters_name_2
            exact_match = var.query_parameters_exact_match_2
          }]
          headers = [{
            name        = var.headers_name_1
            exact_match = var.headers_exact_match_1
            }, {
            name        = var.headers_name_2
            exact_match = var.headers_exact_match_2
            }, {
            name = var.headers_name_3
          }]
          path = {
            prefix = var.path_prefix_1
          }
          }, {
          path = {
            prefix = var.path_prefix_2
          }
          target_pool = var.target_pool_name_2
        }]
        }, {
        host = var.host_3
        rules = [{
          path = {
            prefix = var.path_prefix_3
          }
          target_pool = var.target_pool_name_3
        }]
      }]
    }
    protocol = var.protocol_http
    }, {
    name = var.listener_name_2
    port = var.listener_port_4
    http = {
      hosts = [{
        host = var.host_4
        rules = [{
          path = {
            prefix = var.path_prefix_4
          }
          target_pool = var.target_pool_name_4
        }]
      }]
    }
    protocol = var.protocol_http
  }]
  networks = [
    {
      network_id = stackit_network.listener_network.network_id
      role       = var.network_role_listeners
    },
    {
      network_id = stackit_network.target_network.network_id
      role       = var.network_role_targets
    }
  ]
  options = {
    ephemeral_address    = var.ephemeral_address
    private_network_only = var.private_network_only
    access_control = {
      allowed_source_ranges = [var.acl]
    }
    observability = {
      logs = {
        credentials_ref = stackit_loadbalancer_observability_credential.observer.credentials_ref
        push_url        = var.observability_logs_push_url
      }
      metrics = {
        credentials_ref = stackit_loadbalancer_observability_credential.observer.credentials_ref
        push_url        = var.observability_metrics_push_url
      }
    }
  }
}
