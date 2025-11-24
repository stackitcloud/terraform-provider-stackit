variable "default_region" {}

provider "stackit" {
  default_region = var.default_region
}

ephemeral "stackit_access_token" "example" {}

provider "echo" {
  data = ephemeral.stackit_access_token.example
}

resource "echo" "example" {
}
