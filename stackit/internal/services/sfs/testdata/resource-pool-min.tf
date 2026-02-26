
variable "project_id" {}
variable "name" {}
variable "availability_zone" {}
variable "performance_class" {}
variable "size_gigabytes" {}
variable "acl" {}

resource "stackit_sfs_resource_pool" "resourcepool" {
  project_id        = var.project_id
  name              = var.name
  availability_zone = var.availability_zone
  performance_class = var.performance_class
  size_gigabytes    = var.size_gigabytes
  ip_acl            = var.acl
}
