resource "stackit_logs_instance" "logs" {
  project_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region         = "eu01"
  display_name   = "logs-instance-example"
  retention_days = 30
}

resource "stackit_logs_instance" "logs2" {
  project_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region         = "eu01"
  display_name   = "logs-instance-example"
  retention_days = 30
  acl = [
    "0.0.0.0/0"
  ]
  description = "Example description"
}
