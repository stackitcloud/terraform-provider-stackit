resource "stackit_authorization_organization_role_assignment" "example" {
  resource_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  role        = "owner"
  subject     = "john.doe@stackit.cloud"
}
