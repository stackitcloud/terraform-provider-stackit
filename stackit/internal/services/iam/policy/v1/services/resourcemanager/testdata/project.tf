variable "organization_id" {}
variable "project_name" {}
variable "project_owner_email" {}
variable "role" {}
variable "subject" {}

resource "stackit_resourcemanager_project" "project" {
  name                = var.project_name
  owner_email         = var.project_owner_email
  parent_container_id = var.organization_id
}

resource "stackit_resourcemanager_project_iam_policy_v1" "iam_policy" {
  resource_id = stackit_resourcemanager_project.project.project_id
  role_bindings = [
    {
      role    = var.role
      subject = var.subject
    }
  ]
}
