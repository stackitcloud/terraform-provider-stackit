variable "project_id" {}
variable "name" {}
variable "instance_version" {}
variable "plan_name" {}

resource "stackit_opensearch_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  version    = var.instance_version
  plan_name  = var.plan_name
}

resource "stackit_opensearch_credential" "credential" {
  project_id  = var.project_id
  instance_id = stackit_opensearch_instance.instance.instance_id
}