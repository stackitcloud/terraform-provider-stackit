variable "organization_id" {}
variable "project_name" {}
variable "service_account_email" {}

resource "stackit_resourcemanager_project" "project" {
  parent_container_id = var.organization_id
  name                = var.project_name
  owner_email         = var.service_account_email
}

resource "stackit_service_enablement" "ske_eu01" {
  project_id = stackit_resourcemanager_project.project.project_id
  service_id = "cloud.stackit.ske"
  region     = "eu01"
}

resource "stackit_service_enablement" "ske_eu02" {
  project_id = stackit_resourcemanager_project.project.project_id
  service_id = "cloud.stackit.ske"
  region     = "eu02"
}
