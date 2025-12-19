# Copyright (c) STACKIT

variable "project_id" {}
variable "name" {}

provider "stackit" {
  test = "test"
}

resource "stackit_network" "network" {
  name       = var.name
  project_id = var.project_id
}