variable "project_id" {}
variable "instance_name" {}
variable "role" {}
variable "subject" {}

resource "stackit_secretsmanager_instance" "instance" {
  project_id = var.project_id
  name       = var.instance_name
}

resource "stackit_secretsmanager_instance_role_binding_v1" "role_binding" {
  resource_id = stackit_secretsmanager_instance.instance.instance_id
  role        = var.role
  subject     = var.subject
}