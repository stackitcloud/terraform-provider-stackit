
resource "stackit_authorization_organization_role_assignment" "serviceaccount" {
  resource_id = var.organization_id
  role        = "organization.member"
  subject     = var.test_service_account
}