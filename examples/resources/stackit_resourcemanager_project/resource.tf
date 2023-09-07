resource "stackit_resourcemanager_project" "example" {
  project_id          = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  parent_container_id = "example-parent-container-abc123"
  name                = "example-container"
  labels = {
    "Label 1" = "foo"
  }
  owner_email = "aa@bb.ccc"
}
