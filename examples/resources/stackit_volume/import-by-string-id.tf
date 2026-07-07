# Only use the import statement, if you want to import an existing volume
import {
  to = stackit_volume.import-example
  id = "${var.project_id},${var.region},${var.volume_id}"
}
