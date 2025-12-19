resource "stackit_sfs_resource_pool" "resourcepool" {
  project_id        = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
  name              = "some-resourcepool"
  availability_zone = "eu01-m"
  performance_class = "Standard"
  size_gigabytes    = 512
  ip_acl = [
    "192.168.42.1/32",
    "192.168.42.2/32"
  ]
  snapshots_are_visible = true
}

# Only use the import statement, if you want to import an existing resource pool
import {
  to = stackit_sfs_resource_pool.resourcepool
  id = "${var.project_id},${var.region},${var.resource_pool_id}"
}