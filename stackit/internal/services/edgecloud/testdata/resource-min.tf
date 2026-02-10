variable "project_id" {}
variable "display_name" {}
variable "plan_id" {}

resource "stackit_edgecloud_instance" "test_instance" {
  project_id   = var.project_id
  display_name = var.display_name
  plan_id      = var.plan_id
}

resource "stackit_edgecloud_kubeconfig" "this" {
  project_id  = var.project_id
  instance_id = stackit_edgecloud_instance.test_instance.instance_id
}

resource "stackit_edgecloud_token" "this" {
  project_id  = var.project_id
  instance_id = stackit_edgecloud_instance.test_instance.instance_id
}

data "stackit_edgecloud_instances" "this" {
  project_id = var.project_id
}

data "stackit_edgecloud_plans" "this" {
  project_id = var.project_id
}
