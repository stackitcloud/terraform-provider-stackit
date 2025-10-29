
variable "project_id" {}
variable "name" {}
variable "quota_id" {}
variable "suspended" {}
variable "region" {}

resource "stackit_scf_organization" "org" {
  project_id = var.project_id
  name       = var.name
  suspended  = var.suspended
  quota_id   = var.quota_id
  region     = var.region
}

resource "stackit_scf_organization_manager" "orgmanager" {
  project_id = var.project_id
  org_id     = stackit_scf_organization.org.org_id
}
data "stackit_scf_platform" "scf_platform" {
  project_id  = var.project_id
  platform_id = stackit_scf_organization.org.platform_id
}