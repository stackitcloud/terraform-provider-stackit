resource "stackit_objectstorage_compliance_lock" "example_bucket" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "stackit_objectstorage_default_retention" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  bucket_name = stackit_objectstorage_compliance_lock.example_bucket.name
  days        = var.retention_days
  mode        = var.retention_mode
}
