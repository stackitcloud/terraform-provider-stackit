# Only use the import statement, if you want to import an existing TelemetryLink
import {
  to = stackit_telemetrylink.import-example
  id = "${var.resource_type},${var.resource_id},${var.region}"
}
