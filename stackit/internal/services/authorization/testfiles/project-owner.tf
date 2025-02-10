
resource "stackit_authorization_project_role_assignment" "serviceaccount_project_owner" {
  resource_id = var.project_id
  role = "owner"
  subject = var.test_service_account
}
