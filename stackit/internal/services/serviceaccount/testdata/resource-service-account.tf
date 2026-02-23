variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

resource "stackit_service_account" "sa" {
  project_id = var.project_id
  name       = var.name
}

resource "stackit_service_account_access_token" "token" {
  project_id            = stackit_service_account.sa.project_id
  service_account_email = stackit_service_account.sa.email
}

resource "stackit_service_account_key" "key" {
  project_id            = stackit_service_account.sa.project_id
  service_account_email = stackit_service_account.sa.email
  ttl_days              = 90
}