terraform {
  required_providers {
    # stackit = {
    #   source  = "registry.terraform.io/stackitcloud/stackit"
    #   version = "~> 0.70"
    # }
    stackitprivatepreview = {
      source  = "registry.terraform.io/mhenselin/stackitprivatepreview"
      version = "~> 0.1"
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
