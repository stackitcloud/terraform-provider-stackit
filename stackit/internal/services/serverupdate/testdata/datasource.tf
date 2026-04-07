data "stackit_server_update_enable" "enable_test" {
  project_id = var.project_id
  server_id  = stackit_server.server.server_id
}

data "stackit_server_update_schedule" "test_schedule" {
  project_id         = var.project_id
  server_id          = stackit_server.server.server_id
  update_schedule_id = stackit_server_update_schedule.test_schedule.update_schedule_id
}

data "stackit_server_update_schedules" "schedules_data_test" {
  project_id = var.project_id
  server_id  = stackit_server.server.server_id
}