# Only use the import statement, if you want to import an existing resourcemanager folder
# Note: There will be a conflict which needs to be resolved manually.
# Must set a configuration value for the owner_email attribute as the provider has marked it as required.
import {
  to = stackit_resourcemanager_folder.import-example
  id = var.container_id
}
