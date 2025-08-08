resource "stackit_objectstorage_credential" "example" {
  project_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  credentials_group_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  expiration_timestamp = "2027-01-02T03:04:05Z"
}

# Only use the import statement, if you want to import an existing objectstorage credential
import {
  to = stackit_objectstorage_credential.import-example
  id = "${var.project_id},${var.region},${var.bucket_credentials_group_id},${var.bucket_credential_id}"
}
