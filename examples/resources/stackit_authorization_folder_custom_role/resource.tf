resource "stackit_resourcemanager_folder" "example" {
  name                = "example_folder"
  owner_email         = "foo.bar@stackit.cloud"
  parent_container_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "stackit_authorization_folder_custom_role" "example" {
  resource_id = stackit_resourcemanager_folder.example.folder_id
  name        = "my.custom.role"
  description = "Some description"
  permissions = [
    "iam.subject.get"
  ]
}

# Only use the import statement, if you want to import an existing custom role
import {
  to = stackit_authorization_folder_custom_role.import-example
  id = "${var.folder_id},${var.custom_role_id}"
}

