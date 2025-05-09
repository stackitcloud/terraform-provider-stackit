variable "project_id" {}
variable "server_id" {}
variable "schedule_name" {}
variable "rrule" {}
variable "enabled" {}
variable "backup_name" {}
variable "retention_period" {}


resource "stackit_server_backup_schedule" "test_schedule" {
  project_id = var.project_id
  server_id  = var.server_id
  name       = var.schedule_name
  rrule      = var.rrule
  enabled    = var.enabled
  backup_properties = {
    name             = var.backup_name
    retention_period = var.retention_period
    volume_ids       = null
  }
}

data "stackit_server_backup_schedule" "schedule_data_test" {
  project_id         = var.project_id
  server_id          = var.server_id
  backup_schedule_id = stackit_server_backup_schedule.test_schedule.backup_schedule_id
}

data "stackit_server_backup_schedules" "schedules_data_test" {
  project_id = var.project_id
  server_id  = var.server_id
}
