resource "stackit_network" "example_with_name" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-with-name"
}

resource "stackit_network" "example_routed_network" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-routed-network"
  labels = {
    "key" = "value"
  }
  routed = true
}

resource "stackit_network" "example_non_routed_network" {
  project_id       = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name             = "example-non-routed-network"
  ipv4_nameservers = ["1.2.3.4", "5.6.7.8"]
  ipv4_gateway     = "10.1.2.3"
  ipv4_prefix      = "10.1.2.0/24"
  labels = {
    "key" = "value"
  }
  routed = false
}

# Only use the import statement, if you want to import an existing network
# Note: There will be a conflict which needs to be resolved manually.
# These attributes cannot be configured together: [ipv4_prefix,ipv4_prefix_length,ipv4_gateway]
import {
  to = stackit_network.import-example
  id = "${var.project_id},${var.network_id}"
}
