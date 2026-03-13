variable "email_regex" {
  type = string
}

data "stackit_service_accounts" "list_regex" {
  project_id  = stackit_service_account.sa.project_id
  email_regex = var.email_regex
}