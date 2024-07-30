resource "stackit_resourcemanager_project" "example" {
  parent_container_id = "example-parent-container-abc123"
  name                = "example-container"
  labels = {
    "Label 1" = "foo"
  }
  members = [
    {
      "subject" : "john.doe@stackit.cloud"
      "role" : "owner",
    },
    {
      "subject" : "some.engineer@stackit.cloud"
      "role" : "reader",
    },
  ]
}
