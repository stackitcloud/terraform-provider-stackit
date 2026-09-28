variable "project_id" {}
variable "server_name" {}
variable "network_name" {}
variable "machine_type" {}
variable "region" {}
variable "availability_zone" {}
variable "script" {}

resource "stackit_network" "network" {
  project_id = var.project_id
  name       = var.network_name
}

resource "stackit_network_interface" "nic" {
  project_id = var.project_id
  network_id = stackit_network.network.network_id
  security   = false
}

data "stackit_image_v2" "ubuntu" {
  project_id = var.project_id
  name       = "Ubuntu 24.04"
}

action "stackit_run_command" "test_action" {
  config {
    project_id            = var.project_id
    server_id             = stackit_server.server.server_id
    region                = var.region
    command_template_name = "RunShellScript"
    parameters = {
      script = var.script
    }
  }
}

resource "stackit_server" "server" {
  project_id        = var.project_id
  name              = var.server_name
  machine_type      = var.machine_type
  availability_zone = var.availability_zone

  boot_volume = {
    source_type           = "image"
    source_id             = data.stackit_image_v2.ubuntu.image_id
    size                  = 32
    delete_on_termination = true
  }

  network_interfaces = [
    stackit_network_interface.nic.network_interface_id
  ]

  agent = {
    provisioning_policy = "ALWAYS"
  }

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.stackit_run_command.test_action]
    }
  }
}
