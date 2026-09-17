resource "stackit_intakes" "example" {
  project_id                              = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  runner_id                               = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  display_name                            = "example-intake"
  description                             = "An example intake for STACKIT Intake service"
  catalog_auth_type                       = "dremio"
  catalog_warehouse                       = "default"
  catalog_uri                             = "https://dremio.eu01.onstackit.cloud/iceberg"
  dremio_token_endpoint                   = "https://dremio.eu01.onstackit.cloud/oauth/token"
  dremio_personal_access_token_wo         = "my-dremio-pat-token"
  dremio_personal_access_token_wo_version = 1

  labels = {
    "env" = "development"
  }
}
