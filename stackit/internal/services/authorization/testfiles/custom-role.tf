
variable "project_id" {}
variable "role_name" {}
variable "role_description" {}
variable "role_permissions_0" {}

resource "stackit_authorization_project_custom_role" "custom-role" {
  resource_id = var.project_id
  name        = var.role_name
  description = var.role_description
  permissions = [var.role_permissions_0]
}
