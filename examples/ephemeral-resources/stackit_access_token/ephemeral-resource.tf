provider "stackit" {
  default_region           = "eu01"
  service_account_key_path = "/path/to/sa_key.json"
  enable_beta_resources    = true
}

ephemeral "stackit_access_token" "example" {}

locals {
  stackit_api_base_url = "https://iaas.api.stackit.cloud"
  public_ip_path       = "/v2/projects/${var.project_id}/regions/${var.region}/public-ips"

  public_ip_payload = {
    labels = {
      key = "value"
    }
  }
}

# Docs: https://registry.terraform.io/providers/magodo/restful/latest
provider "restful" {
  base_url = local.stackit_api_base_url

  security = {
    http = {
      token = {
        token = ephemeral.stackit_access_token.example.access_token
      }
    }
  }
}

resource "restful_resource" "public_ip_restful" {
  path = local.public_ip_path
  body = local.public_ip_payload

  read_path     = "$(path)/$(body.id)"
  update_path   = "$(path)/$(body.id)"
  update_method = "PATCH"
  delete_path   = "$(path)/$(body.id)"
  delete_method = "DELETE"
}

# Docs: https://registry.terraform.io/providers/Mastercard/restapi/latest
provider "restapi" {
  uri                  = local.stackit_api_base_url
  write_returns_object = true

  headers = {
    Authorization = "Bearer ${ephemeral.stackit_access_token.example.access_token}"
    Content-Type  = "application/json"
  }

  create_method  = "POST"
  update_method  = "PATCH"
  destroy_method = "DELETE"
}

resource "restapi_object" "public_ip_restapi" {
  path = local.public_ip_path
  data = jsonencode(local.public_ip_payload)

  id_attribute   = "id"
  read_method    = "GET"
  create_method  = "POST"
  update_method  = "PATCH"
  destroy_method = "DELETE"
}
