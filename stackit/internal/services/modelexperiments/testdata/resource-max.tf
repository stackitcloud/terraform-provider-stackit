
variable "project_id" {}
variable "region" {}

// Instance Variables
variable "name" {}
variable "description" {}
variable "deleted_experiment_retention" {}
variable "label_value" {}

// Token Variables
variable "token_name" {}
variable "token_description" {}
variable "ttl_duration" {}
variable "token_label_value" {}

// Instance Resources
resource "stackit_modelexperiments_instance" "example" {
  project_id                   = var.project_id
  region                       = var.region
  name                         = var.name
  description                  = var.description
  deleted_experiment_retention = var.deleted_experiment_retention
  labels = {
    label = var.label_value
  }
}

data "stackit_modelexperiments_instance" "example" {
  project_id  = var.project_id
  region      = var.region
  instance_id = stackit_modelexperiments_instance.example.instance_id
}

// Token Resources
resource "stackit_modelexperiments_token" "example" {
  project_id   = var.project_id
  region       = var.region
  instance_id  = stackit_modelexperiments_instance.example.instance_id
  name         = var.token_name
  description  = var.token_description
  ttl_duration = var.ttl_duration
  labels = {
    label = var.token_label_value
  }
}

data "stackit_modelexperiments_token" "example" {
  project_id  = var.project_id
  region      = var.region
  instance_id = stackit_modelexperiments_instance.example.instance_id
  token_id    = stackit_modelexperiments_token.example.token_id
}