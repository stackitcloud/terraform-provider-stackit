variable "project_id" {}
variable "region" {}
variable "display_name" {}
variable "plan_id" {}
variable "description" {}
variable "expiration" {}
variable "recreate_before" {}

resource "stackit_edgecloud_instance" "test_instance" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
  plan_id      = var.plan_id
  description  = var.description
}

resource "stackit_edgecloud_kubeconfig" "by_name" {
  project_id      = var.project_id
  instance_name   = stackit_edgecloud_instance.test_instance.display_name
  expiration      = var.expiration
  recreate_before = var.recreate_before
}

resource "stackit_edgecloud_kubeconfig" "by_id" {
  project_id      = var.project_id
  instance_id     = stackit_edgecloud_instance.test_instance.instance_id
  expiration      = var.expiration
  recreate_before = var.recreate_before
}

resource "stackit_edgecloud_token" "by_name" {
  project_id      = var.project_id
  instance_name   = stackit_edgecloud_instance.test_instance.display_name
  expiration      = var.expiration
  recreate_before = var.recreate_before
}

resource "stackit_edgecloud_token" "by_id" {
  project_id      = var.project_id
  instance_id     = stackit_edgecloud_instance.test_instance.instance_id
  expiration      = var.expiration
  recreate_before = var.recreate_before
}

data "stackit_edgecloud_instances" "this" {
  project_id = var.project_id
}


data "stackit_edgecloud_plans" "this" {
  project_id = var.project_id
}
