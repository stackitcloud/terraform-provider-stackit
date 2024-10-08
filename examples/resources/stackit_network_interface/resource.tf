resource "stackit_network_interface" "example" {
  project_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  allowed_addresses = ["1.2.3.4"]
  security_groups   = ["xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"]
}