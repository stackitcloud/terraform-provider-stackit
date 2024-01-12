resource "stackit_ske_cluster" "example" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name               = "example"
  kubernetes_version = "1.25"
  node_pools = [
    {
      name               = "np-example"
      machine_type       = "x.x"
      os_version         = "x.x.x"
      minimum            = "2"
      maximum            = "3"
      availability_zones = ["eu01-3"]
    }
  ]
  maintenance = {
    enable_kubernetes_version_updates    = true
    enable_machine_image_version_updates = true
    start                                = "01:00:00Z"
    end                                  = "02:00:00Z"
  }
}
