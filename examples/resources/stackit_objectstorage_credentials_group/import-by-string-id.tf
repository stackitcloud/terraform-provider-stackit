# Only use the import statement, if you want to import an existing objectstorage credential group
import {
  to = stackit_objectstorage_credentials_group.import-example
  id = "${var.project_id},${var.region},${var.bucket_credentials_group_id}"
}
