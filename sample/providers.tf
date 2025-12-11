terraform {
  required_providers {
    stackitalpha = {
      source = "registry.terraform.io/stackitcloud/stackitalpha"
      version = "~> 0.1"
    }
  }
}

provider "stackitalpha" {
  default_region        = "eu01"
  enable_beta_resources = true
  service_account_key_path = "./service_account.json"
}
