ephemeral "stackit_access_token" "example" {}

// https://registry.terraform.io/providers/Mastercard/restapi/latest/docs
provider "restapi" {
  uri                  = "https://iaas.api.eu01.stackit.cloud/"
  write_returns_object = true

  headers = {
    Authorization = "Bearer ${ephemeral.stackit_access_token.example.access_token}"
    Content-Type  = "application/json"
  }

  create_method  = "POST"
  update_method  = "PUT"
  destroy_method = "DELETE"
}

resource "restapi_object" "iaas_keypair" {
  path = "/v2/keypairs"

  data = jsonencode({
    labels = {
      key = "testvalue"
    }
    name      = "test-keypair-123"
    publicKey = file(chomp("~/.ssh/id_rsa.pub"))
  })

  id_attribute   = "name"
  read_method    = "GET"
  create_method  = "POST"
  update_method  = "PATCH"
  destroy_method = "DELETE"
}
