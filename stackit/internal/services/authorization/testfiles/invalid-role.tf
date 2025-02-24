
resource "stackit_authorization_project_role_assignment" "invalid_role" {
  resource_id = var.project_id
  role        = "thisrolesdoesnotexist"
  subject     = var.test_service_account
}
