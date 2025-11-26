variable "default_region" {}

provider "stackit" {
  default_region        = var.default_region
  enable_beta_resources = true
}

ephemeral "stackit_access_token" "example" {}

provider "echo" {
  data = ephemeral.stackit_access_token.example
}

resource "echo" "example" {
}
