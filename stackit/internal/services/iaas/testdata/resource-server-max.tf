variable "project_id" {}
variable "name" {}
variable "name_not_updated" {}
variable "machine_type" {}
variable "image_id" {}
variable "availability_zone" {}
variable "label" {}
variable "user_data" {}

variable "policy" {}
variable "size" {}
variable "public_key" {}
variable "service_account_mail" {}

resource "stackit_affinity_group" "affinity_group" {
  project_id = var.project_id
  name       = var.name_not_updated
  policy     = var.policy
}

resource "stackit_volume" "base_volume" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  size              = var.size
  source = {
    id   = var.image_id
    type = "image"
  }
}

resource "stackit_volume" "data_volume" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  size              = var.size
}

resource "stackit_server_volume_attach" "data_volume_attachment" {
  project_id = var.project_id
  server_id  = stackit_server.server.server_id
  volume_id  = stackit_volume.data_volume.volume_id
}

resource "stackit_network" "network" {
  project_id = var.project_id
  name       = var.name
}

resource "stackit_network_interface" "network_interface_init" {
  project_id = var.project_id
  network_id = stackit_network.network.network_id
}

resource "stackit_network_interface" "network_interface_second" {
  project_id = var.project_id
  network_id = stackit_network.network.network_id
}

resource "stackit_server_network_interface_attach" "network_interface_second_attachment" {
  project_id           = var.project_id
  network_interface_id = stackit_network_interface.network_interface_second.network_interface_id
  server_id            = stackit_server.server.server_id
}

resource "stackit_key_pair" "key_pair" {
  name       = var.name_not_updated
  public_key = var.public_key
}

resource "stackit_server_service_account_attach" "attached_service_account" {
  project_id            = var.project_id
  server_id             = stackit_server.server.server_id
  service_account_email = var.service_account_mail
}

resource "stackit_server" "server" {
  project_id         = var.project_id
  name               = var.name
  machine_type       = var.machine_type
  affinity_group     = stackit_affinity_group.affinity_group.affinity_group_id
  availability_zone  = var.availability_zone
  keypair_name       = stackit_key_pair.key_pair.name
  network_interfaces = [stackit_network_interface.network_interface_init.network_interface_id]
  user_data          = var.user_data
  boot_volume = {
    source_type = "volume"
    source_id   = stackit_volume.base_volume.volume_id
  }
  labels = {
    "acc-test" : var.label
  }
}
