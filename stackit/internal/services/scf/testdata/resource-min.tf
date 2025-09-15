
variable "project_id" {}
variable "name" {}

resource "stackit_scf_organization" "org" {
  project_id = var.project_id
  name       = var.name
}

resource "stackit_scf_organization_manager" "orgmanager" {
  project_id = var.project_id
  org_id     = stackit_scf_organization.org.org_id
}