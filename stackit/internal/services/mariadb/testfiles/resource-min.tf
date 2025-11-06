variable "project_id" {}
variable "name" {}
variable "db_version" {}
variable "plan_name" {}

resource "stackit_mariadb_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  version    = var.db_version
  plan_name  = var.plan_name
}

resource "stackit_mariadb_credential" "credential" {
  project_id  = var.project_id
  instance_id = stackit_mariadb_instance.instance.instance_id
}