# Only use the import statement, if you want to import an existing TelemetryRouter instance
import {
  to = stackit_telemetryrouter_instance.import-example
  id = "${var.project_id},${var.region},${var.router_instance_id}"
}
