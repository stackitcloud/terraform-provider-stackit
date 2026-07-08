resource "stackit_network_interface" "example" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  allowed_addresses  = ["192.168.0.0/24"]
  security_group_ids = ["xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"]
}