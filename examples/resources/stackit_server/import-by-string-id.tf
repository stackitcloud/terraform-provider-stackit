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
