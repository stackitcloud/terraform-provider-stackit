# Only use the import statement, if you want to import an existing objectstorage bucket
import {
  to = stackit_objectstorage_bucket.import-example
  id = "${var.project_id},${var.region},${var.bucket_name}"
}
