# Only use the import statement, if you want to import an existing objectstorage default retention
import {
  to = stackit_objectstorage_default_retention.import-example
  id = "${var.project_id},${var.region},${var.bucket_name}"
}
