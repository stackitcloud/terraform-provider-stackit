# Only use the import statement, if you want to import an existing objectstorage credential
import {
  to = stackit_objectstorage_credential.import-example
  id = "${var.project_id},${var.region},${var.bucket_credentials_group_id},${var.bucket_credential_id}"
}
