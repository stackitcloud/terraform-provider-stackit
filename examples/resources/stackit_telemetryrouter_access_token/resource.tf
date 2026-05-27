resource "stackit_telemetryrouter_access_token" "accessToken" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "telemetryrouter-access-token-example"
}

resource "stackit_telemetryrouter_access_token" "accessToken2" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "telemetryrouter-access-token-example"
  ttl          = 30
  description  = "Example description"
}

# Only use the import statement, if you want to import an existing TelemetryRouter access token
# Note: The generated access token is only available upon creation.
import {
  to = stackit_telemetryrouter_access_token.import-example
  id = "${var.project_id},${var.region},${var.telemetryrouter_instance_id},${var.telemetryrouter_access_token_id}"
}