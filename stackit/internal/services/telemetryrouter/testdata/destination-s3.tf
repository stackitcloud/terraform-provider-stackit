
variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "description" {}
variable "config_filter_key" {}
variable "config_filter_level" {}
variable "config_filter_matcher" {}
variable "config_filter_value0" {}
variable "config_filter_value1" {}
variable "config_s3_bucket" {}
variable "objectstorage_credentials_group_name" {}

resource "stackit_objectstorage_bucket" "bucket" {
  project_id = var.project_id
  name       = var.config_s3_bucket
}

resource "stackit_objectstorage_credentials_group" "credentials_group" {
  project_id = var.project_id
  name       = var.objectstorage_credentials_group_name
}

resource "stackit_objectstorage_credential" "credential" {
  project_id           = stackit_objectstorage_credentials_group.credentials_group.project_id
  credentials_group_id = stackit_objectstorage_credentials_group.credentials_group.credentials_group_id
}

resource "stackit_telemetryrouter_instance" "router" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
}

resource "stackit_telemetryrouter_destination" "destination" {
  project_id   = var.project_id
  region       = var.region
  instance_id  = stackit_telemetryrouter_instance.router.instance_id
  display_name = var.display_name
  description  = var.description
  config = {
    filter = {
      attributes = [
        {
          key     = var.config_filter_key
          level   = var.config_filter_level
          matcher = var.config_filter_matcher
          values = [
            var.config_filter_value0,
            var.config_filter_value1
          ]
        }
      ]
    }
    config_type = "S3"
    s3 = {
      access_key = {
        id     = stackit_objectstorage_credential.credential.access_key
        secret = stackit_objectstorage_credential.credential.secret_access_key
      }
      bucket   = var.config_s3_bucket
      endpoint = join("/", slice(split("/", stackit_objectstorage_bucket.bucket.url_path_style), 0, 3))
    }
  }
}
