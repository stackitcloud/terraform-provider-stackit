variable "project_id" {}
variable "runner_name" {}
variable "intake_name" {}
variable "max_message_size_kib" {}
variable "max_messages_per_hour" {}

variable "dremio_display_name" {}
variable "dremio_user_email" {}
variable "dremio_user_first_name" {}
variable "dremio_user_last_name" {}
variable "dremio_user_name" {}
variable "dremio_user_password" {}
variable "dremio_personal_access_token" {}

resource "stackit_intake_runner" "example" {
  project_id            = var.project_id
  name                  = var.runner_name
  max_message_size_kib  = var.max_message_size_kib
  max_messages_per_hour = var.max_messages_per_hour
}

resource "stackit_dremio_instance" "dremio" {
  project_id   = var.project_id
  display_name = var.dremio_display_name
  authentication = {
    type = "local-only"
  }
}

resource "stackit_dremio_user" "dremio_user" {
  project_id  = var.project_id
  instance_id = stackit_dremio_instance.dremio.instance_id

  email      = var.dremio_user_email
  first_name = var.dremio_user_first_name
  last_name  = var.dremio_user_last_name
  name       = var.dremio_user_name
  password   = var.dremio_user_password
}

resource "stackit_intakes" "example" {
  project_id                   = var.project_id
  runner_id                    = stackit_intake_runner.example.runner_id
  name                         = var.intake_name
  catalog_auth_type            = "dremio"
  catalog_warehouse            = "default"
  catalog_uri                  = startswith(stackit_dremio_instance.dremio.endpoints.catalog, "https://") ? stackit_dremio_instance.dremio.endpoints.catalog : "https://${stackit_dremio_instance.dremio.endpoints.catalog}"
  dremio_token_endpoint        = startswith(stackit_dremio_instance.dremio.endpoints.ui, "https://") ? "${stackit_dremio_instance.dremio.endpoints.ui}/oauth/token" : "https://${stackit_dremio_instance.dremio.endpoints.ui}/oauth/token"
  dremio_personal_access_token = var.dremio_personal_access_token
}
