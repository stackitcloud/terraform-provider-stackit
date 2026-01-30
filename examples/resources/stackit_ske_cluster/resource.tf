resource "stackit_ske_cluster" "example" {
  project_id             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name                   = "example"
  kubernetes_version_min = "x.x"
  node_pools = [
    {
      name               = "np-example"
      machine_type       = "x.x"
      os_version         = "x.x.x"
      os_name            = "xxx"
      minimum            = "2"
      maximum            = "3"
      availability_zones = ["eu01-3"]
      volume_type        = "storage_premium_perf6"
      volume_size        = "48"
    }
  ]
  maintenance = {
    enable_kubernetes_version_updates    = true
    enable_machine_image_version_updates = true
    start                                = "01:00:00Z"
    end                                  = "02:00:00Z"
  }
}

# Only use the import statement, if you want to import an existing ske cluster
import {
  to = stackit_ske_cluster.import-example
  id = "${var.project_id},${var.region},${var.ske_name}"
}