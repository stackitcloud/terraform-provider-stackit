variable "project_id" {}
variable "server_name" {}
variable "schedule_name" {}
variable "rrule" {}
variable "enabled" {}
variable "maintenance_window" {}
variable "server_id" {}

resource "stackit_server_update_schedule" "test_schedule" {
  project_id         = var.project_id
  server_id          = var.server_id
  name               = var.schedule_name
  rrule              = var.rrule
  enabled            = var.enabled
  maintenance_window = var.maintenance_window
}

data "stackit_server_update_schedule" "test_schedule" {
  project_id         = var.project_id
  server_id          = var.server_id
  update_schedule_id = stackit_server_update_schedule.test_schedule.update_schedule_id
}

data "stackit_server_update_schedules" "schedules_data_test" {
  project_id = var.project_id
  server_id  = var.server_id
}
