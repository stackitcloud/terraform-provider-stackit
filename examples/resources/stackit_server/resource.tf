resource "stackit_server" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-server"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "7b10e105-295b-4369-b6e0-567ec940a02b" // Ubuntu 24.04
  }
  availability_zone = "xxxx-x"
  machine_type      = "g2i.1"
  network_interfaces = [
    stackit_network_interface.example.network_interface_id
  ]
}