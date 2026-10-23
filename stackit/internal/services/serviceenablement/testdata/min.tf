variable "organization_id" {}
variable "project_name" {}
variable "service_account_email" {}
variable "service_id" {}

# NOTE: This testcase is considered complete. It tests the service enablement resource
# implementation for the ske service in a single region. When onboarding new services,
# only add them to the "Max" test. There is no need to add them here.

resource "stackit_resourcemanager_project" "project" {
  parent_container_id = var.organization_id
  name                = var.project_name
  owner_email         = var.service_account_email
}

resource "stackit_service_enablement" "service_enablement" {
  project_id = stackit_resourcemanager_project.project.project_id
  service_id = var.service_id
}

