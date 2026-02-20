
variable "project_id" {}
variable "name" {}
variable "region" {}

resource "stackit_intake_runner" "example" {
  project_id            = var.project_id
  name                  = var.name
  region                = var.region
  description           = "An example runner for Intake"
  max_message_size_kib  = 1024
  max_messages_per_hour = 1100
  labels = {
    "created_by" = "terraform-provider-stackit"
    "env"        = "development"
  }
}