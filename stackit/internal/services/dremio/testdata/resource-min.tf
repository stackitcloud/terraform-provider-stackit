
variable "project_id"{}
variable "region" {}
variable "display_name" {}
variable "authentication_type" {}

resource "stackit_dremio_instance" "example" {
    project_id = var.project_id
    region = var.region
    display_name = var.display_name
    authentication = {
        type = var.authentication_type
    }
}

data "stackit_dremio_instance" "example" {
  project_id    = var.project_id
  region        = var.region
  instance_id   = stackit_dremio_instance.example.instance_id
}