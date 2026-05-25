
variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "description" {}
variable "config_filter_key" {}
variable "config_filter_level" {}
variable "config_filter_matcher" {}
variable "config_filter_value0" {}
variable "config_filter_value1" {}

resource "stackit_telemetryrouter_instance" "router" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
}

resource "stackit_telemetryrouter_instance" "destination" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
}

resource "stackit_telemetryrouter_access_token" "accessToken" {
  project_id   = var.project_id
  instance_id  = stackit_telemetryrouter_instance.destination.instance_id
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
      bearer_token = stackit_telemetryrouter_access_token.accessToken.access_token
      uri          = "https;//${stackit_telemetryrouter_instance.destination.uri}/v1/logs"
    }
  }
}
