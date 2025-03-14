
variable "project_id" {}
variable "test_service_account" {}
variable "organization_id" {}

resource "stackit_authorization_project_role_assignment" "serviceaccount" {
  resource_id = var.project_id
  role        = "reader"
  subject     = var.test_service_account
}
