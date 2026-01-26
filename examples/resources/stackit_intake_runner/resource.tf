resource "stackit_intake_runner" "example" {
  project_id            = var.project_id
  name                  = "example-runner-full"
  description           = "An example runner for STACKIT Intake"
  max_message_size_kib  = 2048
  max_messages_per_hour = 1500
  labels = {
    "created_by" = "terraform-example"
    "env"        = "production"
  }
  region = var.region
}

import {
  to = stackit_intake_runner.example
  id = "${var.project_id},${var.region},${var.runner_id}"
}