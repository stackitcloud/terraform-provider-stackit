variable "project_id" {}
variable "name" {}
variable "acl" {}
variable "access_scope" {}
variable "backup_schedule" {}
variable "flavor_id" {}
variable "storage_class" {}
variable "storage_size" {}
variable "instance_version" {}
variable "retention_days" {}
variable "flavor_cpu" {}
variable "flavor_ram" {}
variable "replicas" {}
variable "region" {}

resource "stackit_postgresflex_instance" "with_flavor_id" {
  project_id = var.project_id
  name       = var.name
  network = {
    acl          = [var.acl]
    access_scope = var.access_scope
  }
  backup_schedule = var.backup_schedule
  flavor_id       = var.flavor_id
  storage = {
    class = var.storage_class
    size  = var.storage_size
  }
  version        = var.instance_version
  retention_days = var.retention_days
}

resource "stackit_postgresflex_instance" "with_flavor" {
  project_id      = var.project_id
  name            = var.name
  acl             = [var.acl]
  backup_schedule = var.backup_schedule
  flavor = {
    cpu = var.flavor_cpu
    ram = var.flavor_ram
  }
  replicas = var.replicas
  storage = {
    class = var.storage_class
    size  = var.storage_size
  }
  version        = var.instance_version
  retention_days = var.retention_days
  region         = var.region
}
