data "stackit_ske_machine_image_versions" "example" {
  version_state = "SUPPORTED"
}

locals {
  flatcar_supported_version = one(flatten([
    for mi in data.stackit_ske_machine_image_versions.example.machine_images : [
      for v in mi.versions :
      v.version
      if mi.name == "flatcar" # or ubuntu
    ]
  ]))
}

resource "stackit_ske_cluster" "example" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name               = "example"
  kubernetes_version = "x.x"
  node_pools = [
    {
      name               = "np-example"
      machine_type       = "x.x"
      os_version         = local.flatcar_supported_version
      os_name            = "flatcar"
      minimum            = "2"
      maximum            = "3"
      availability_zones = ["eu01-1"]
      volume_type        = "storage_premium_perf6"
      volume_size        = "48"
    }
  ]
}