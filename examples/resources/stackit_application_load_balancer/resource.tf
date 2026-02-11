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

variable "availability_zone" {
  description = "An availability zone"
  type        = string
  default     = "eu01-1"
}

variable "machine_type" {
  description = "The machine flavor with 2GB of RAM and 1 core"
  type        = string
  default     = "c2i.1"
}

variable "label_key" {
  description = "An optional label key"
  type        = string
  default     = "key"
}

variable "label_value" {
  description = "An optional label value"
  type        = string
  default     = "value"
}

# Create a network
resource "stackit_network" "network" {
  project_id       = var.project_id
  name             = "example-network"
  ipv4_nameservers = ["1.1.1.1"]
  ipv4_prefix      = "192.168.2.0/25"
  routed           = true
}

# Create a network interface
resource "stackit_network_interface" "nic" {
  project_id = var.project_id
  network_id = stackit_network.network.network_id
  lifecycle {
    ignore_changes = [
      security_group_ids,
    ]
  }
}

# Create a key pair for accessing the target server instance
resource "stackit_key_pair" "keypair" {
  name       = "example-key-pair"
  public_key = chomp(file("path/to/id_rsa.pub"))
}

# Create a target server instance
resource "stackit_server" "server" {
  project_id        = var.project_id
  name              = "example-server"
  machine_type      = var.machine_type
  keypair_name      = stackit_key_pair.keypair.name
  availability_zone = var.availability_zone

  boot_volume = {
    size                  = 20
    source_type           = "image"
    source_id             = var.image_id
    delete_on_termination = true
  }

  network_interfaces = [
    stackit_network_interface.nic.network_interface_id
  ]

  # Explicit dependencies to ensure ordering
  depends_on = [
    stackit_network.network,
    stackit_key_pair.keypair,
    stackit_network_interface.nic
  ]
}

# Create example credentials for observability of the ALB
# Create real credentials in your stackit observability
resource "stackit_loadbalancer_observability_credential" "observability" {
  project_id   = var.project_id
  display_name = "my-cred"
  password     = "password"
  username     = "username"
}

# Create a Application Load Balancer
resource "stackit_application_load_balancer" "example" {
  project_id = var.project_id
  region     = "eu01"
  name       = "example-load-balancer"
  plan_id    = "p10"
  // Hint: Automatically create an IP for the ALB lifecycle by setting ephemeral_address = true or use:
  // external_address = "124.124.124.124"
  labels = {
    (var.label_key) = var.label_value
  }
  listeners = [{
    name = "my-listener"
    port = 443
    http = {
      hosts = [{
        host = "*"
        rules = [{
          target_pool = "my-target-pool"
          web_socket  = true
          query_parameters = [{
            name        = "my-query-key"
            exact_match = "my-query-value"
          }]
          headers = [{
            name        = "my-header-key"
            exact_match = "my-header-value"
          }]
          path = {
            prefix = "/path"
          }
          cookie_persistence = {
            name = "my-cookie"
            ttl  = "60s"
          }
        }]
      }]
    }
    https = {
      certificate_config = {
        certificate_ids = [
          # Currently no TF provider available, needs to be added with API
          # https://docs.api.stackit.cloud/documentation/certificates/version/v2
          "name-v1-8c81bd317af8a03b8ef0851ccb074eb17d1ad589b540446244a5e593f78ef820"
        ]
      }
    }
    protocol = "PROTOCOL_HTTPS"
    # Currently no TF provider available, needs to be added with API
    # https://docs.api.stackit.cloud/documentation/alb-waf/version/v1alpha
    waf_config_name = "my-waf-config"
    }
  ]
  networks = [
    {
      network_id = stackit_network.network.network_id
      role       = "ROLE_LISTENERS_AND_TARGETS"
    }
  ]
  options = {
    acl                  = ["123.123.123.123/24", "12.12.12.12/24"]
    ephemeral_address    = true
    private_network_only = false
    observability = {
      logs = {
        credentials_ref = stackit_loadbalancer_observability_credential.observability.credentials_ref
        push_url        = "https://logs.stackit<id>.argus.eu01.stackit.cloud/instances/<instance-id>/loki/api/v1/push"
      }
      metrics = {
        credentials_ref = stackit_loadbalancer_observability_credential.observability.credentials_ref
        push_url        = "https://push.metrics.stackit<id>.argus.eu01.stackit.cloud/instances/<instance-id>/api/v1/receive"
      }
    }
  }
  target_pools = [
    {
      name = "my-target-pool"
      active_health_check = {
        interval            = "0.500s"
        interval_jitter     = "0.010s"
        timeout             = "1s"
        healthy_threshold   = "5"
        unhealthy_threshold = "3"
        http_health_checks = {
          ok_status = ["200", "201"]
          path      = "/healthy"
        }
      }
      target_port = 80
      targets = [
        {
          display_name = "my-target"
          ip           = stackit_network_interface.nic.ipv4
        }
      ]
      tls_config = {
        enabled                     = true
        skip_certificate_validation = false
        custom_ca                   = chomp(file("path/to/PEM_formated_CA"))
      }
    }
  ]
  disable_target_security_group_assignment = false # only needed if targets are not in the same network
}
