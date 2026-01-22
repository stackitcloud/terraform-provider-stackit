# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: Apache-2.0

terraform {
  required_providers {
    # stackit = {
    #   source  = "registry.terraform.io/stackitcloud/stackit"
    #   version = "~> 0.70"
    # }
    stackitprivatepreview = {
      source  = "tfregistry.sysops.stackit.rocks/mhenselin/stackitprivatepreview"
      version = "0.0.0-SNAPSHOT-e91e10e"
    }
  }
}

# provider "stackit" {
#   default_region           = "eu01"
#   enable_beta_resources    = true
#   service_account_key_path = "./service_account.json"
# }

provider "stackitprivatepreview" {
  default_region           = "eu01"
  enable_beta_resources    = true
  service_account_key_path = "../service_account.json"
}
