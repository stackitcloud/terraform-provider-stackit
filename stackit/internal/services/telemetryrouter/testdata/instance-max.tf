
variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "description" {}
variable "filter_key" {}
variable "filter_level" {}
variable "filter_matcher" {}
variable "filter_value0" {}
variable "filter_value1" {}

resource "stackit_telemetryrouter_instance" "router" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
  description  = var.description
  filter = {
    attributes = [
      {
        key     = var.filter_key
        level   = var.filter_level
        matcher = var.filter_matcher
        values = [
          var.filter_value0,
          var.filter_value1
        ]
      }
    ]
  }
}
