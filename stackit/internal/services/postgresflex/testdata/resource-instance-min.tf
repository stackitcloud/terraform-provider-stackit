variable "project_id" {}
variable "name" {}
variable "acl" {}
variable "backup_schedule" {}
variable "flavor_id" {}
variable "storage_class" {}
variable "storage_size" {}
variable "instance_version" {}

resource "stackit_postgresflex_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  network = {
    acl = [var.acl]
  }
  backup_schedule = var.backup_schedule
  flavor_id       = var.flavor_id
  storage = {
    class = var.storage_class
    size  = var.storage_size
  }
  version = var.instance_version
}
