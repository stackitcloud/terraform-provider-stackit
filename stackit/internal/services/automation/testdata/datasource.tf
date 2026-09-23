data "stackit_volume_automation" "test_data" {
  project_id    = var.project_id
  automation_id = stackit_volume_automation.test.automation_id
}
