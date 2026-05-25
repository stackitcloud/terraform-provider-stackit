
variable "project_id" {}
variable "resource_type" {}
variable "resource_id" {}
variable "region" {}
variable "display_name" {}
variable "description" {}

resource "stackit_telemetryrouter_instance" "router" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
}

resource "stackit_telemetryrouter_access_token" "accessToken" {
  project_id   = var.project_id
  instance_id  = stackit_telemetryrouter_instance.router.instance_id
  region       = var.region
  display_name = var.display_name
}

resource "stackit_telemetrylink" "link" {
  resource_type       = var.resource_type
  resource_id         = var.resource_id
  region              = var.region
  display_name        = var.display_name
  description         = var.description
  access_token        = var.access_token
  telemetry_router_id = var.telemetry_router_id
}
