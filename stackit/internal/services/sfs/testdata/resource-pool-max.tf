
variable "project_id" {}
variable "region" {}
variable "name" {}
variable "availability_zone" {}
variable "performance_class" {}
variable "size_gigabytes" {}
variable "ip_acl_1" {}
variable "ip_acl_2" {}
variable "snapshots_are_visible" {}

resource "stackit_sfs_resource_pool" "resourcepool" {
  project_id        = var.project_id
  region            = var.region
  name              = var.name
  availability_zone = var.availability_zone
  performance_class = var.performance_class
  size_gigabytes    = var.size_gigabytes
  ip_acl = [
    var.ip_acl_1,
    var.ip_acl_2
  ]
  snapshots_are_visible = var.snapshots_are_visible
}
