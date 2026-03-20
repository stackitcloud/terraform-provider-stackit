variable "project_id" {}
variable "regions" {}
variable "backend_bucket_type" {}
variable "region" {}
variable "optimizer" {}

# object storage
variable "bucket_name" {}
variable "credentials_name" {}

resource "stackit_objectstorage_bucket" "bucket" {
  project_id = var.project_id
  name       = var.bucket_name
}

resource "stackit_objectstorage_credentials_group" "group" {
  project_id = var.project_id
  name       = var.credentials_name
}

resource "stackit_objectstorage_credential" "creds" {
  project_id           = var.project_id
  credentials_group_id = stackit_objectstorage_credentials_group.group.credentials_group_id
}

# cdn
resource "stackit_cdn_distribution" "distribution" {
  project_id = var.project_id
  config = {
    optimizer = {
      enabled = var.optimizer
    }
    regions = var.regions
    backend = {
      type       = var.backend_bucket_type
      bucket_url = "https://${stackit_objectstorage_bucket.bucket.name}.object.storage.eu01.onstackit.cloud"
      credentials = {
        access_key_id     = stackit_objectstorage_credential.creds.access_key
        secret_access_key = stackit_objectstorage_credential.creds.secret_access_key
      }
      region = var.region
    }
  }
}

data "stackit_cdn_distribution" "bucket_ds" {
  project_id      = var.project_id
  distribution_id = stackit_cdn_distribution.distribution.distribution_id
}
