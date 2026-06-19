variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "instance_version" {}

resource "stackit_workflows_instance" "workflow" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
  version      = var.instance_version

  identity_provider = {
    type = "stackit"
  }
}
