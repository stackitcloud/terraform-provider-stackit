
variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "retention_days" {}
variable "acl" {}
variable "description" {}

resource "stackit_logs_instance" "logs" {
  project_id     = var.project_id
  region         = var.region
  display_name   = var.display_name
  retention_days = var.retention_days
  acl = [
    var.acl
  ]
  description = var.description
}
