resource "stackit_dremio_instance" "example" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "exampleName"
  description  = "Example description"
  authentication = {
    type = "local-only" // "oauth" or "azuread" for IDP config

    oauth = { // only needed if "oauth" is given as type
      authority_url = "authority"
      client_id     = "client-id"
      client_secret = "client-secret"
      jwt_claims = {
        user_name = "example"
      }
      scope = "idp-scope"
      parameters = [
        { "name" : "example", "value" : "example-value" }
      ]
    }

    azuread = { // only needed if "azuread" is given as type
      authority_url = "authority"
      client_id     = "client-id"
      client_secret = "client-secret"
    }
  }
}

# Only use the import statement, if you want to import an existing dns zone
import {
  to = stackit_dremio_instance.import_example
  id = "${var.project_id},${var.region},${var.instance_id}"
}