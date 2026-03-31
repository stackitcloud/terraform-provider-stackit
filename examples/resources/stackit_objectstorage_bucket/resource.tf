resource "stackit_objectstorage_bucket" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-bucket"
}

## With compliance lock
resource "stackit_objectstorage_compliance_lock" "example_with_lock" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "stackit_objectstorage_bucket" "example_with_lock" {
  depends_on  = [stackit_objectstorage_compliance_lock.example_with_lock]
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "example-bucket-with-lock"
  object_lock = true
}


# Only use the import statement, if you want to import an existing objectstorage bucket
import {
  to = stackit_objectstorage_bucket.import-example
  id = "${var.project_id},${var.region},${var.bucket_name}"
}
