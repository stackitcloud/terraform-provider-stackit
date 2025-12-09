
variable "project_id" {}
variable "test_service_account" {}
variable "organization_id" {}

resource "stackit_authorization_project_role_assignment" "serviceaccount" {
  resource_id = var.project_id
  role        = "reader"
  subject     = var.test_service_account
}

resource "stackit_resourcemanager_folder" "test" {
  name                = "test"
  owner_email         = "foo.bar@stackit.cloud"
  parent_container_id = var.organization_id
}