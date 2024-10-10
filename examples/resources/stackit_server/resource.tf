resource "stackit_server" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "my-server"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  initial_networking = {
    network_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    security_group_ids = ["xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"]
  }
  availability_zone = "eu01-1"
  labels = {
    "key" = "value"
  }
  machine_type = "t1.1"
  keypair_name = "my-key-pair-name"
}