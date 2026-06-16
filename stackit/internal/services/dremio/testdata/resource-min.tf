
variable "project_id" {}
variable "region" {}

// Instance Variables
variable "display_name" {}
variable "authentication_type" {}

// User Variables
variable "email" {}
variable "first_name" {}
variable "last_name" {}
variable "name" {}
variable "password" {}

// Instance Resources
resource "stackit_dremio_instance" "example" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
  authentication = {
    type = var.authentication_type
  }
}

data "stackit_dremio_instance" "example" {
  project_id  = var.project_id
  region      = var.region
  instance_id = stackit_dremio_instance.example.instance_id
}

// User Resources
resource "stackit_dremio_user" "example" {
  project_id  = var.project_id
  region      = var.region
  instance_id = stackit_dremio_instance.example.instance_id

  email      = var.email
  first_name = var.first_name
  last_name  = var.last_name
  name       = var.name
  password   = var.password
}

data "stackit_dremio_user" "example" {
  project_id  = var.project_id
  region      = var.region
  instance_id = stackit_dremio_instance.example.instance_id
  user_id     = stackit_dremio_user.example.user_id
}