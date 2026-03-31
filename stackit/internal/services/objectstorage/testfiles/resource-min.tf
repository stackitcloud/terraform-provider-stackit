
variable "project_id" {}
variable "objectstorage_bucket_name" {}
variable "objectstorage_credentials_group_name" {}
variable "expiration_timestamp" {}

variable "objectstorage_bucket_name_with_lock" {}
variable "object_lock" {}

resource "stackit_objectstorage_bucket" "bucket" {
  project_id = var.project_id
  name       = var.objectstorage_bucket_name
}

resource "stackit_objectstorage_credentials_group" "credentials_group" {
  project_id = var.project_id
  name       = var.objectstorage_credentials_group_name
}

resource "stackit_objectstorage_credential" "credential" {
  project_id           = stackit_objectstorage_credentials_group.credentials_group.project_id
  credentials_group_id = stackit_objectstorage_credentials_group.credentials_group.credentials_group_id
}

resource "stackit_objectstorage_credential" "credential_time" {
  project_id           = stackit_objectstorage_credentials_group.credentials_group.project_id
  credentials_group_id = stackit_objectstorage_credentials_group.credentials_group.credentials_group_id
  expiration_timestamp = var.expiration_timestamp
}

resource "stackit_objectstorage_compliance_lock" "compliance_lock" {
  project_id = var.project_id
}

resource "stackit_objectstorage_bucket" "bucket_object_lock" {
  depends_on  = [stackit_objectstorage_compliance_lock.compliance_lock]
  project_id  = var.project_id
  name        = var.objectstorage_bucket_name_with_lock
  object_lock = var.object_lock
}