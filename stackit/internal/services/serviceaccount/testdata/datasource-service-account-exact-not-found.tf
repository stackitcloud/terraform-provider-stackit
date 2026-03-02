variable "not_found_email" {
  type = string
}

data "stackit_service_account" "sa_not_found" {
  project_id = stackit_service_account.sa.project_id
  email      = var.not_found_email
}