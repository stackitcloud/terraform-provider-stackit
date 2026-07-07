resource "stackit_resourcemanager_folder" "example" {
  name        = "example_folder"
  owner_email = "foo.bar@stackit.cloud"
  # in this case a org-id
  parent_container_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "stackit_authorization_folder_role_assignment" "fra" {
  resource_id = stackit_resourcemanager_folder.example.folder_id
  role        = "reader"
  subject     = "foo.bar@stackit.cloud"
}