
variable "resource_type" {}
variable "resource_id" {}
variable "region" {}
variable "display_name" {}
variable "description" {}
variable "access_token" {}
variable "telemetry_router_id" {}

resource "stackit_telemetrylink" "link" {
  resource_type       = var.resource_type
  resource_id         = var.resource_id
  region              = var.region
  display_name        = var.display_name
  description         = var.description
  access_token        = var.access_token
  telemetry_router_id = var.telemetry_router_id
}
