
variable "project_id" {}
variable "region" {}

// Instance Variables
variable "name" {}

// Token Variables
variable "token_name" {}

// Instance Resources
resource "stackit_modelexperiments_instance" "example" {
  project_id = var.project_id
  region     = var.region
  name       = var.name
}

data "stackit_modelexperiments_instance" "example" {
  project_id  = var.project_id
  region      = var.region
  instance_id = stackit_modelexperiments_instance.example.instance_id
}

// Token Resources
resource "stackit_modelexperiments_token" "example" {
  project_id  = var.project_id
  region      = var.region
  instance_id = stackit_modelexperiments_instance.example.instance_id
  name        = var.token_name
}
