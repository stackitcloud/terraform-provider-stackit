resource "stackit_network" "example" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name               = "example-network"
  nameservers        = ["1.2.3.4", "5.6.7.8"]
  ipv4_prefix_length = 24
  labels = {
    "key" = "value"
  }
  routed = true
}
