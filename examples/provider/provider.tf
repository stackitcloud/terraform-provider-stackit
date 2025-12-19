# Copyright (c) STACKIT

provider "stackitprivatepreview" {
  default_region = "eu01"
}

# Authentication

# Token flow (scheduled for deprecation and will be removed on December 17, 2025)
provider "stackitprivatepreview" {
  default_region        = "eu01"
  service_account_token = var.service_account_token
}

# Key flow
provider "stackitprivatepreview" {
  default_region      = "eu01"
  service_account_key = var.service_account_key
  private_key         = var.private_key
}

# Key flow (using path)
provider "stackitprivatepreview" {
  default_region           = "eu01"
  service_account_key_path = var.service_account_key_path
  private_key_path         = var.private_key_path
}

