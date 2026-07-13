variable "project_id" {}
variable "name" {}
variable "acl1" {}
variable "flavor_id" {}
variable "storage_class" {}
variable "storage_size" {}
variable "access_scope" {}
variable "retention_days" {}
variable "backup_schedule" {}
variable "username" {}
variable "role" {}
variable "server_version" {}
variable "region" {}

resource "stackit_sqlserverflex_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  flavor_id  = var.flavor_id
  storage = {
    class = var.storage_class
    size  = var.storage_size
  }
  network = {
    acl          = [var.acl1]
    access_scope = var.access_scope
  }
  retention_days  = var.retention_days
  version         = var.server_version
  backup_schedule = var.backup_schedule
  region          = var.region
}

resource "stackit_sqlserverflex_user" "user" {
  project_id  = stackit_sqlserverflex_instance.instance.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
  username    = var.username
  roles       = [var.role]
}

data "stackit_sqlserverflex_instance" "instance" {
  project_id  = var.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
}

data "stackit_sqlserverflex_user" "user" {
  project_id  = var.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
  user_id     = stackit_sqlserverflex_user.user.user_id
}
