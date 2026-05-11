resource "stackit_telemetrylink_link" "link" {
  resource_type       = "project"
  resource_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region              = "eu01"
  display_name        = "telemetrylink-example"
  access_token        = "eyJxxx"
  telemetry_router_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "stackit_telemetrylink_link" "link2" {
  resource_type       = "project"
  resource_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region              = "eu01"
  display_name        = "telemetrylink-example"
  description         = "telemetrylink description"
  access_token        = "eyJxxx"
  telemetry_router_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# Only use the import statement, if you want to import an existing TelemetryLink link
import {
  to = stackit_telemetrylink_link.import-example
  id = "${var.resource_type},${var.resource_id},${var.region}"
}
