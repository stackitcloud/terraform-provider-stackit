resource "stackit_resourcemanager_folder" "example" {
  name                = "example-folder"
  owner_email         = "foo.bar@stackit.cloud"
  parent_container_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# Note:
# You can add projects under folders.
# However, when deleting a project, be aware:
#   - Projects may remain "invisible" for up to 7 days after deletion
#   - During this time, deleting the parent folder may fail because the project is still technically linked
resource "stackit_resourcemanager_project" "example_project" {
  name                = "example-project"
  owner_email         = "foo.bar@stackit.cloud"
  parent_container_id = stackit_resourcemanager_folder.example.container_id
}