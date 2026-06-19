variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "instance_version" {}
variable "idp_name" {}
variable "idp_client_id" {}
variable "idp_client_secret" {}
variable "idp_scope" {}
variable "idp_discovery_endpoint" {}
variable "bundle_name" {}
variable "bundle_url" {}
variable "bundle_branch" {}
variable "bundle_username" {}
variable "bundle_password" {}
variable "bundle_subdir" {} // ignored — subdir intentionally omitted to test PATCH-clearing

resource "stackit_workflows_instance" "workflow" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
  version      = var.instance_version

  identity_provider = {
    type               = "oauth2"
    name               = var.idp_name
    client_id          = var.idp_client_id
    client_secret      = var.idp_client_secret
    scope              = var.idp_scope
    discovery_endpoint = var.idp_discovery_endpoint
  }
}

resource "stackit_workflows_dag_bundle" "bundle" {
  project_id  = var.project_id
  region      = var.region
  instance_id = stackit_workflows_instance.workflow.instance_id

  name = var.bundle_name
  git = {
    url    = var.bundle_url
    branch = var.bundle_branch
    auth = {
      type     = "basic"
      username = var.bundle_username
      password = var.bundle_password
    }
  }
}
