
variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "description" {}
variable "config_filter_key" {}
variable "config_filter_level" {}
variable "config_filter_matcher" {}
variable "config_filter_value0" {}
variable "config_filter_value1" {}
variable "plan_name" {}
variable "grafana_admin_enabled" {}

resource "stackit_observability_instance" "instance" {
  project_id            = var.project_id
  name                  = var.display_name
  plan_name             = var.plan_name
  grafana_admin_enabled = var.grafana_admin_enabled
}

resource "stackit_observability_credential" "credential" {
  project_id  = var.project_id
  instance_id = stackit_observability_instance.instance.instance_id
}

resource "stackit_telemetryrouter_instance" "router" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
}

resource "stackit_telemetryrouter_destination" "destination" {
  project_id   = var.project_id
  region       = var.region
  instance_id  = stackit_telemetryrouter_instance.router.instance_id
  display_name = var.display_name
  description  = var.description
  config = {
    filter = {
      attributes = [
        {
          key     = var.config_filter_key
          level   = var.config_filter_level
          matcher = var.config_filter_matcher
          values = [
            var.config_filter_value0,
            var.config_filter_value1
          ]
        }
      ]
    }
    config_type = "OpenTelemetry"
    opentelemetry = {
      basic_auth = {
        username = stackit_observability_credential.credential.username
        password = stackit_observability_credential.credential.password
      }
      uri = stackit_observability_instance.instance.otlp_http_logs_url
    }
  }
}
