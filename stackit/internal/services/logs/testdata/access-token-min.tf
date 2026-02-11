variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "retention_days" {}
variable "permissions" {}

resource "stackit_logs_instance" "logs" {
  project_id     = var.project_id
  region         = var.region
  display_name   = var.display_name
  retention_days = var.retention_days
}

resource "stackit_logs_access_token" "accessToken" {
  project_id   = var.project_id
  instance_id  = stackit_logs_instance.logs.instance_id
  region       = var.region
  display_name = var.display_name
  permissions = [
    var.permissions
  ]
}

