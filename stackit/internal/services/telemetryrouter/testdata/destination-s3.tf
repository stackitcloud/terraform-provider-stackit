
variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "description" {}
variable "config_filter_key" {}
variable "config_filter_level" {}
variable "config_filter_matcher" {}
variable "config_filter_value0" {}
variable "config_filter_value1" {}
variable "config_s3_id" {}
variable "config_s3_secret" {}
variable "config_s3_bucket" {}
variable "config_s3_endpoint" {}

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
    config_type = "S3"
    s3 = {
      access_key = {
        id     = var.config_s3_id
        secret = var.config_s3_secret
      }
      bucket   = var.config_s3_bucket
      endpoint = var.config_s3_endpoint
    }
  }
}
