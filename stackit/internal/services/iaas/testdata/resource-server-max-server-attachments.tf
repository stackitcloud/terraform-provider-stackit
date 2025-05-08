resource "stackit_server_volume_attach" "data_volume_attachment" {
  project_id = var.project_id
  server_id  = stackit_server.server.server_id
  volume_id  = stackit_volume.data_volume.volume_id
}

resource "stackit_server_network_interface_attach" "network_interface_second_attachment" {
  project_id           = var.project_id
  network_interface_id = stackit_network_interface.network_interface_second.network_interface_id
  server_id            = stackit_server.server.server_id
}
