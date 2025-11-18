data "stackit_ske_availability_zones" "example" {
  region = "eu01"
}

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
      availability_zones = data.stackit_ske_availability_zones.example.availability_zones
      volume_type        = "storage_premium_perf6"
      volume_size        = "48"
    }
  ]
}