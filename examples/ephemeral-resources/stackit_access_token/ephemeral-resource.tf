ephemeral "stackit_access_token" "example" {}

// https://registry.terraform.io/providers/Mastercard/restapi/latest/docs
provider "restapi" {
  alias                = "stackit_iaas"
  uri                  = "https://iaas.api.eu01.stackit.cloud"
  write_returns_object = true

  headers = {
    "Authorization" = "Bearer ${ephemeral.stackit_access_token.example.access_token}"
  }

  create_method  = "GET"
  update_method  = "GET"
  destroy_method = "GET"
}