resource "stackit_telemetryrouter_instance" "router" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "router-instance"
}

resource "stackit_telemetryrouter_instance" "router2" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "router-instance"
  description  = "Example description"
  filter = {
    attributes = [
      {
        key     = "key"
        level   = "logRecord"
        matcher = "!="
        values  = ["test1", "test2"]
      },
      {
        key     = "key2"
        level   = "resource"
        matcher = "="
        values  = ["test3"]
      }
    ]
  }
}

# Only use the import statement, if you want to import an existing TelemetryRouter instance
import {
  to = stackit_telemetryrouter_instance.import-example
  id = "${var.project_id},${var.region},${var.router_instance_id}"
}
