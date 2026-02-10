resource "stackit_server" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-server"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "59838a89-51b1-4892-b57f-b3caf598ee2f" // Ubuntu 24.04
  }
  availability_zone = "xxxx-x"
  machine_type      = "g2i.1"
  network_interfaces = [
    stackit_network_interface.example.network_interface_id
  ]
}

# Only use the import statement, if you want to import an existing server
# Note: There will be a conflict which needs to be resolved manually.
# Must set a configuration value for the boot_volume.source_type and boot_volume.source_id attribute as the provider has marked it as required.
# Since those attributes are not fetched in general from the API call, after adding them this would replace your server resource after an terraform apply.
# In order to prevent this you need to add:
# lifecycle {
#   ignore_changes = [ boot_volume ]
# }
import {
  to = stackit_server.import-example
  id = "${var.project_id},${var.region},${var.server_id}"
}