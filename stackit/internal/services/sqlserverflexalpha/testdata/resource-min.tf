# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: Apache-2.0

variable "project_id" {}
variable "name" {}
variable "flavor_cpu" {}
variable "flavor_ram" {}
variable "username" {}
variable "role" {}

resource "stackit_sqlserverflex_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  flavor = {
    cpu = var.flavor_cpu
    ram = var.flavor_ram
  }
}

resource "stackit_sqlserverflex_user" "user" {
  project_id  = stackit_sqlserverflex_instance.instance.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
  username    = var.username
  roles       = [var.role]
}

data "stackit_sqlserverflex_instance" "instance" {
  project_id  = var.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
}

data "stackit_sqlserverflex_user" "user" {
  project_id  = var.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
  user_id     = stackit_sqlserverflex_user.user.user_id
}
