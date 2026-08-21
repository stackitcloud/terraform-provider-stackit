resource "stackit_telemetrylink" "link" {
  resource_type           = "project"
  resource_id             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region                  = "eu01"
  display_name            = "telemetrylink-example"
  access_token_wo         = "eyJxxx"
  access_token_wo_version = 1
  telemetry_router_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# access_token is kept for backwards compatibility, but access_token_wo (see above) should be preferred
# since it is never persisted to the Terraform state.
resource "stackit_telemetrylink" "link2" {
  resource_type       = "project"
  resource_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region              = "eu01"
  display_name        = "telemetrylink-example"
  description         = "telemetrylink description"
  access_token        = "eyJxxx"
  telemetry_router_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
