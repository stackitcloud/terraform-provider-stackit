
resource "stackit_authorization_project_role_assignment" "serviceaccount_duplicate" {
  resource_id = var.project_id
  role        = "reader"
  subject     = var.test_service_account
}
