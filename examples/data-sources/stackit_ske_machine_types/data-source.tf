data "stackit_ske_machine_types" "example" {}

locals {
  matched_machine = [
    for machine in data.stackit_ske_machine_types.example.machine_types : machine.name
    if machine.cpu == 8 && machine.memory == 16
  ]
}

resource "stackit_ske_cluster" "example" {
  project_id             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name                   = "example"
  kubernetes_version_min = "x.x"
  node_pools = [
    {
      name               = "np-example"
      machine_type       = local.matched_machine[0]
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