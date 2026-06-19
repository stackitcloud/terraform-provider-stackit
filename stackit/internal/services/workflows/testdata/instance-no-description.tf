variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "description" {} // ignored — description intentionally omitted from the resource to test PATCH-clearing
variable "instance_version" {}
variable "idp_name" {}
variable "idp_client_id" {}
variable "idp_client_secret" {}
variable "idp_scope" {}
variable "idp_discovery_endpoint" {}

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
