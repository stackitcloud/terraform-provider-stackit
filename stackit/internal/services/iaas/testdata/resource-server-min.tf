variable "project_id" {}
variable "name" {}
variable "machine_type" {}
variable "image_id" {}


resource "stackit_server" "server" {
  project_id   = var.project_id
  name         = var.name
  machine_type = var.machine_type
  boot_volume = {
    source_type           = "image"
    size                  = 16
    source_id             = var.image_id
    delete_on_termination = true
  }
}
