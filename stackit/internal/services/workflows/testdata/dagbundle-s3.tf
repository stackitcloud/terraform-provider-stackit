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
variable "bucket_name" {}
variable "endpoint" {}
variable "prefix" {}
variable "access_key_id" {}
variable "secret_access_key" {}

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
  s3 = {
    bucket_name = var.bucket_name
    endpoint    = var.endpoint
    prefix      = var.prefix
    auth = {
      type              = "access_key"
      access_key_id     = var.access_key_id
      secret_access_key = var.secret_access_key
    }
  }
}
