data "stackit_server_backup_enable" "enable_test" {
  project_id = var.project_id
  server_id  = stackit_server.server.server_id
}

data "stackit_server_backup_schedule" "schedule_data_test" {
  project_id         = var.project_id
  server_id          = stackit_server.server.server_id
  backup_schedule_id = stackit_server_backup_schedule.test_schedule.backup_schedule_id
}

data "stackit_server_backup_schedules" "schedules_data_test" {
  project_id = var.project_id
  server_id  = stackit_server.server.server_id
}
