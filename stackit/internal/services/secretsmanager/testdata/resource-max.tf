variable "project_id" {}
variable "instance_name" {}
variable "user_description" {}
variable "write_enabled" {}
variable "acl1" {}
variable "acl2" {}

resource "stackit_secretsmanager_instance" "instance" {
  project_id = var.project_id
  name       = var.instance_name
  acls = [
    var.acl1,
    var.acl2,
  ]
}

resource "stackit_secretsmanager_user" "user" {
  project_id    = var.project_id
  instance_id   = stackit_secretsmanager_instance.instance.instance_id
  description   = var.user_description
  write_enabled = var.write_enabled
}


data "stackit_secretsmanager_instance" "instance" {
  project_id  = var.project_id
  instance_id = stackit_secretsmanager_instance.instance.instance_id
}

data "stackit_secretsmanager_user" "user" {
  project_id  = var.project_id
  instance_id = stackit_secretsmanager_instance.instance.instance_id
  user_id     = stackit_secretsmanager_user.user.user_id
}
