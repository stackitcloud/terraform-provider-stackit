
variable "project_id" {}
variable "name" {}
variable "platform_id" {}
variable "quota_id" {}
variable "suspended" {}
variable "region" {}

resource "stackit_scf_organization" "org" {
  project_id  = var.project_id
  name        = var.name
  platform_id = var.platform_id
  quota_id    = var.quota_id
  suspended   = var.suspended
}

resource "stackit_scf_organization_manager" "orgmanager" {
  project_id = var.project_id
  org_id     = stackit_scf_organization.org.org_id
}
data "stackit_scf_platform" "scf_platform" {
  project_id = var.project_id
  guid       = stackit_scf_organization.org.platform_id
}