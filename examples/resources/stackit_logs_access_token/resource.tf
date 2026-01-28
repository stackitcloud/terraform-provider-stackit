resource "stackit_logs_access_token" "accessToken" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "logs-access-token-example"
  permissions = [
    "read"
  ]
}

resource "stackit_logs_access_token" "accessToken2" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "logs-access-token-example"
  lifetime     = 30
  permissions = [
    "write"
  ]
  description = "Example description"
}

# Only use the import statement, if you want to import an existing logs instance
import {
  to = stackit_logs_access_token.import-example
  id = "${var.project_id},${var.region},${var.logs_instance_id},${var.logs_access_token_id}"
}
