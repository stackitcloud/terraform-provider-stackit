
variable "project_id" {}
variable "resource_type" {}
variable "resource_id" {}
variable "region" {}
variable "display_name" {}

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
  access_token        = stackit_telemetryrouter_access_token.accessToken.access_token
  telemetry_router_id = stackit_telemetryrouter_instance.router.instance_id
}
