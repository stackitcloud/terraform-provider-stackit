
variable "project_id" {}
variable "resource_type" {}
variable "resource_id" {}
variable "region" {}
variable "display_name" {}
variable "description" {}
variable "access_token_wo_version" {}

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
  resource_type = var.resource_type
  resource_id   = var.resource_id
  region        = var.region
  display_name  = var.display_name
  description   = var.description
  # in the MIN test we use the legacy field, in the MAX test the write-only field
  access_token_wo         = stackit_telemetryrouter_access_token.accessToken.access_token
  access_token_wo_version = var.access_token_wo_version
  telemetry_router_id     = stackit_telemetryrouter_instance.router.instance_id
}
