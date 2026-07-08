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