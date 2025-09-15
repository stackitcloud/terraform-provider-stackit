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

# Only use the import statement, if you want to import an existing resourcemanager folder
# Note: There will be a conflict which needs to be resolved manually.
# Must set a configuration value for the owner_email attribute as the provider has marked it as required.
import {
  to = stackit_resourcemanager_folder.import-example
  id = var.container_id
}