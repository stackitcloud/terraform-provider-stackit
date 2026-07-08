# Only use the import statement, if you want to import an existing server volume attachment
import {
  to = stackit_server_volume_attach.import-example
  id = "${var.project_id},${var.region},${var.server_id},${var.volume_id}"
}
