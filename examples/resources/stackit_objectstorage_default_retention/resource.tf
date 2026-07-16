resource "stackit_objectstorage_compliance_lock" "example_lock" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "stackit_objectstorage_bucket" "bucket_object_lock" {
  depends_on  = [stackit_objectstorage_compliance_lock.example_lock]
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "example-bucket-with-lock"
  object_lock = true
}

resource "stackit_objectstorage_default_retention" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  bucket_name = stackit_objectstorage_bucket.bucket_object_lock.name
  days        = 2
  mode        = "GOVERNANCE"
}
