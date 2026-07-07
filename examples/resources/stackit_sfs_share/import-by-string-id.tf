# Only use the import statement, if you want to import an existing sfs share
import {
  to = stackit_sfs_resource_pool.resourcepool
  id = "${var.project_id},${var.region},${var.resource_pool_id},${var.share_id}"
}
