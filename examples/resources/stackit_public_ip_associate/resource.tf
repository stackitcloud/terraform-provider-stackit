resource "stackit_public_ip_associate" "example" {
  project_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  public_ip_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_interface_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# Only use the import statement, if you want to import an existing public ip associate
import {
  to = stackit_public_ip_associate.import-example
  id = "${var.project_id},${var.public_ip_id},${var.network_interface_id}"
}
