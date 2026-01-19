
variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "retention_days" {}

resource "stackit_logs_instance" "logs" {
  project_id     = var.project_id
  region         = var.region
  display_name   = var.display_name
  retention_days = var.retention_days
}
