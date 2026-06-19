data "stackit_workflows_provider_options" "options" {
  region = "eu01"
}

resource "stackit_workflows_instance" "instance" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "my-instance"
  version      = data.stackit_workflows_provider_options.options.versions.0.version

  identity_provider = {
    type               = "oauth2"
    name               = "azure"
    client_id          = "xxx"
    client_secret      = "xxx"
    scope              = "openid email"
    discovery_endpoint = "https://login.microsoftonline.com/.../v2.0/.well-known/openid-configuration"
  }
}
