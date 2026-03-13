variable "email_suffix" {
  type = string
}

data "stackit_service_accounts" "list_suffix" {
  project_id   = stackit_service_account.sa.project_id
  email_suffix = var.email_suffix
}