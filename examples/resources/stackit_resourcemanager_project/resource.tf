resource "stackit_resourcemanager_project" "example" {
  parent_container_id = "example-parent-container-abc123"
  name                = "example-container"
  labels = {
    "Label 1" = "foo"
    // "networkArea" = stackit_network_area.foo.network_area_id
  }
  owner_email = "john.doe@stackit.cloud"
}

# Only use the import statement, if you want to import an existing resourcemanager project
# Note: There will be a conflict which needs to be resolved manually.
# Must set a configuration value for the owner_email attribute as the provider has marked it as required.
import {
  to = stackit_resourcemanager_project.import-example
  id = var.container_id
}
