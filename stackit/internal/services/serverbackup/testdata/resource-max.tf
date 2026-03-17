variable "project_id" {}
variable "schedule_name" {}
variable "rrule" {}
variable "enabled" {}
variable "backup_name" {}
variable "retention_period" {}
variable "region" {}

# server
variable "server_name" {}
variable "network_name" {}
variable "machine_type" {}
variable "image_id" {}

# create server
resource "stackit_network" "network" {
  project_id = var.project_id
  name       = var.network_name
}

resource "stackit_network_interface" "nic" {
  project_id = var.project_id
  network_id = stackit_network.network.network_id
}

resource "stackit_server" "server" {
  project_id   = var.project_id
  name         = var.server_name
  machine_type = var.machine_type
  boot_volume = {
    source_type           = "image"
    size                  = 16
    source_id             = var.image_id
    delete_on_termination = true
  }
  network_interfaces = [
    stackit_network_interface.nic.network_interface_id
  ]
}

resource "stackit_server_backup_schedule" "test_schedule" {
  project_id = var.project_id
  server_id  = stackit_server.server.server_id
  name       = var.schedule_name
  rrule      = var.rrule
  enabled    = var.enabled
  backup_properties = {
    name             = var.backup_name
    retention_period = var.retention_period
    volume_ids       = null
  }
  region = var.region
}

data "stackit_server_backup_schedule" "schedule_data_test" {
  project_id         = var.project_id
  server_id          = stackit_server.server.server_id
  backup_schedule_id = stackit_server_backup_schedule.test_schedule.backup_schedule_id
}

data "stackit_server_backup_schedules" "schedules_data_test" {
  project_id = var.project_id
  server_id  = stackit_server.server.server_id
}
