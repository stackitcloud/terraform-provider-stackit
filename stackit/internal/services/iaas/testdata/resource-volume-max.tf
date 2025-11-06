variable "project_id" {}
variable "availability_zone" {}
variable "name" {}
variable "size" {}
variable "description" {}
variable "performance_class" {}
variable "label" {}

resource "stackit_volume" "volume_size" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  name              = var.name
  size              = var.size
  description       = var.description
  performance_class = var.performance_class
  labels = {
    "acc-test" : var.label
  }
}

resource "stackit_volume" "volume_source" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  name              = var.name
  description       = var.description
  performance_class = var.performance_class
  size              = var.size
  source = {
    id   = stackit_volume.volume_size.volume_id
    type = "volume"
  }
  labels = {
    "acc-test" : var.label
  }
}