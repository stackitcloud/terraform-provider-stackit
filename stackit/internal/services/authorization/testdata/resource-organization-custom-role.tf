
variable "organization_id" {}
variable "role_name" {}
variable "role_description" {}
variable "role_permissions_0" {}

resource "stackit_authorization_organization_custom_role" "organization_custom_role" {
  resource_id = var.organization_id
  name        = var.role_name
  description = var.role_description
  permissions = [var.role_permissions_0]
}
