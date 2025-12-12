resource "stackit_server_volume_attach" "attached_volume" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  volume_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# Only use the import statement, if you want to import an existing server volume attachment
import {
  to = stackit_server_volume_attach.import-example
  id = "${var.project_id},${var.region},${var.server_id},${var.volume_id}"
}