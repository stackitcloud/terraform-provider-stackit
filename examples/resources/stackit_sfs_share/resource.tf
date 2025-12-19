resource "stackit_sfs_share" "example" {
  project_id                 = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
  resource_pool_id           = "YYYYYYYY-YYYY-YYYY-YYYY-YYYYYYYYYYYY"
  name                       = "my-nfs-share"
  export_policy              = "high-performance-class"
  space_hard_limit_gigabytes = 32
}

# Only use the import statement, if you want to import an existing sfs share
import {
  to = stackit_sfs_resource_pool.resourcepool
  id = "${var.project_id},${var.region},${var.resource_pool_id},${var.share_id}"
}