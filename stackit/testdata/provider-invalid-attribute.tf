# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: Apache-2.0

variable "project_id" {}
variable "name" {}

provider "stackit" {
  test = "test"
}

resource "stackit_network" "network" {
  name       = var.name
  project_id = var.project_id
}