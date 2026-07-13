resource "stackit_authorization_organization_custom_role" "example" {
  resource_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "my.custom.role"
  description = "Some description"
  permissions = [
    "iam.subject.get"
  ]
}