resource "stackit_public_ip" "example" {
  project_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_interface_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  labels = {
    "key" = "value"
  }
}

# Only use the import statement, if you want to import an existing public ip
import {
  to = stackit_public_ip.import-example
  id = "${var.project_id},${var.region},${var.public_ip_id}"
}