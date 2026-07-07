# Only use the import statement, if you want to import an existing image
# Must set a configuration value for the local_file_path attribute as the provider has marked it as required.
# Since this attribute is not fetched in general from the API call, after adding it this would replace your image resource after an terraform apply.
# In order to prevent this you need to add:
#lifecycle {
#    ignore_changes = [ local_file_path ]
#  }
import {
  to = stackit_image.import-example
  id = "${var.project_id},${var.region},${var.image_id}"
}
