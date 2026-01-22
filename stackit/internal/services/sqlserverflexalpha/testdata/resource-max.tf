# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: Apache-2.0

variable "project_id" {}
variable "name" {}
variable "acl1" {}
variable "flavor_cpu" {}
variable "flavor_ram" {}
variable "storage_class" {}
variable "storage_size" {}
variable "options_retention_days" {}
variable "backup_schedule" {}
variable "username" {}
variable "role" {}
variable "server_version" {}
variable "region" {}

resource "stackit_sqlserverflex_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  acl        = [var.acl1]
  flavor = {
    cpu = var.flavor_cpu
    ram = var.flavor_ram
  }
  storage = {
    class = var.storage_class
    size  = var.storage_size
  }
  version = var.server_version
  options = {
    retention_days = var.options_retention_days
  }
  backup_schedule = var.backup_schedule
  region          = var.region
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
