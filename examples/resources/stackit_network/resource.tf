resource "stackit_network" "example" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name               = "example-network"
  ipv4_nameservers   = ["1.2.3.4", "5.6.7.8"]
  ipv4_prefix_length = 24
  ipv4_gateway       = "10.1.2.1"
  ipv4_prefix        = "10.1.2.0/24"
  labels = {
    "key" = "value"
  }
  routed = true
}
resource "stackit_network" "example_ipv6" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name               = "example-network"
  ipv6_nameservers   = ["2001:4860:4860::8888", "2001:4860:4860::8844"]
  ipv6_prefix_length = 56
  ipv6_gateway       = "10.1.2.1"
  ipv6_prefix        = "10.1.2.0/24"
  labels = {
    "key" = "value"
  }
  routed = true
}
