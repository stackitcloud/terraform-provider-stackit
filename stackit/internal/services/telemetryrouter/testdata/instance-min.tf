
variable "project_id" {}
variable "region" {}
variable "display_name" {}

resource "stackit_telemetryrouter_instance" "router" {
  project_id     = var.project_id
  region         = var.region
  display_name   = var.display_name
}
