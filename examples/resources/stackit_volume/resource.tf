resource "stackit_volume" "example" {
  project_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name              = "my_volume"
  availability_zone = "eu01-1"
  size              = 64
  labels = {
    "key" = "value"
  }
}

# Only use the import statement, if you want to import an existing volume
import {
  to = stackit_volume.import-example
  id = "${var.project_id},${var.region},${var.volume_id}"
}