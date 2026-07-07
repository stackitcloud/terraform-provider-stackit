# Only use the import statement, if you want to import an existing TelemetryRouter destination
import {
  to = stackit_telemetryrouter_destination.import-example
  id = "${var.project_id},${var.region},${var.telemetryrouter_instance_id},${var.destination_id}"
}
