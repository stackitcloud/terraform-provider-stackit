resource "stackit_resourcemanager_project" "example" {
  parent_container_id = "example-parent-container-abc123"
  name                = "example-container"
  labels = {
    "Label 1" = "foo"
    // "networkArea" = stackit_network_area.foo.network_area_id
  }
  owner_email = "john.doe@stackit.cloud"
}
