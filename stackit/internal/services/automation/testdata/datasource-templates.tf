variable "project_id" {}

data "stackit_automation_templates" "templates" {
  project_id = var.project_id
}
