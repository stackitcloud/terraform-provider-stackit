
variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "description" {}
variable "config_filter_key" {}
variable "config_filter_level" {}
variable "config_filter_matcher" {}
variable "config_filter_value0" {}
variable "config_filter_value1" {}
variable "config_opentelemetry_bearer_token" {}
variable "config_opentelemetry_uri" {}

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
      bearer_token = var.config_opentelemetry_bearer_token
      uri          = var.config_opentelemetry_uri
    }
  }
}
