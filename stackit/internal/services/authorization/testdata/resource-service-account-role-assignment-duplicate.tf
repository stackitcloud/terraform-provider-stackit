variable "name" {}
variable "act_as_name" {}
variable "project_id" {}
variable "role" {}

resource "stackit_service_account" "iam" {
  project_id = var.project_id
  name       = var.name
}

resource "stackit_authorization_project_role_assignment" "pr_sa" {
  resource_id = var.project_id
  role        = "editor"
  subject     = stackit_service_account.iam.email
}

resource "stackit_service_account" "act_as" {
  project_id = var.project_id
  name       = var.act_as_name
}

resource "stackit_authorization_service_account_role_assignment" "sa" {
  resource_id = stackit_service_account.iam.service_account_id
  role        = var.role
  subject     = stackit_service_account.act_as.email
}

# Duplicate resource to trigger the validation error
resource "stackit_authorization_service_account_role_assignment" "sa_dup" {
  resource_id = stackit_service_account.iam.service_account_id
  role        = var.role
  subject     = stackit_service_account.act_as.email
}
