resource "stackit_resourcemanager_project" "example" {
  parent_container_id = "example-parent-container-abc123"
  name                = "example-container"
  labels = {
    "Label 1" = "foo"
    # "networkArea" = stackit_network_area.foo.network_area_id

    # optional - you can set a billing reference by adding a label with the 'billingReference' key
    "billingReference" = "fooref"
  }
  owner_email = "john.doe@stackit.cloud"
}
