variable "role" {}
variable "subject" {}
variable "parent_container_id" {}

resource "stackit_authorization_organization_role_assignment" "ora" {
  resource_id = var.parent_container_id
  role        = var.role
  subject     = var.subject
}