resource "stackit_intake_runner" "example" {
  project_id            = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name                  = "example-runner"
  region                = "eu01"
  description           = "An example runner for STACKIT Intake"
  max_message_size_kib  = 1024
  max_messages_per_hour = 1000
  labels = {
    "env" = "development"
  }
}
