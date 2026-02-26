
variable "project_id" {}
variable "resource_pool_name" {}
variable "export_policy_name" {}
variable "name" {}
variable "space_hard_limit_gigabytes" {}

resource "stackit_sfs_resource_pool" "resourcepool" {
  project_id        = var.project_id
  name              = var.resource_pool_name
  availability_zone = "eu01-m"
  performance_class = "Standard"
  size_gigabytes    = 512
  ip_acl            = ["192.168.42.1/32"]
  region            = "eu01"
}

resource "stackit_sfs_export_policy" "exportpolicy" {
  project_id = var.project_id
  name       = var.export_policy_name
  rules = [
    {
      ip_acl = ["192.168.2.0/24"]
      order  = 1
    }
  ]
}

resource "stackit_sfs_share" "share" {
  project_id                 = var.project_id
  resource_pool_id           = stackit_sfs_resource_pool.resourcepool.resource_pool_id
  name                       = var.name
  export_policy              = stackit_sfs_export_policy.exportpolicy.name
  space_hard_limit_gigabytes = var.space_hard_limit_gigabytes
}
