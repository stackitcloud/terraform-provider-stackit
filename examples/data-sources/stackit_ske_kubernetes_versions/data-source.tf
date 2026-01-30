data "stackit_ske_kubernetes_versions" "example" {
  version_state = "SUPPORTED"
}

resource "stackit_ske_cluster" "example" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name               = "example"
  kubernetes_version = data.stackit_ske_kubernetes_versions.example.kubernetes_versions.0.version
  node_pools = [
    {
      name               = "np-example"
      machine_type       = "x.x"
      os_version         = "x.x.x"
      os_name            = "xxx"
      minimum            = "2"
      maximum            = "3"
      availability_zones = ["eu01-1"]
      volume_type        = "storage_premium_perf6"
      volume_size        = "48"
    }
  ]
}