resource "stackit_workflows_instance" "minimal" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "my-workflows"
  version      = "workflows-3.0-airflow-3.1"

  # OAuth2 identity provider is required — the STACKIT IdP variant is not yet
  # accepted by the backend.
  identity_provider = {
    type               = "oauth2"
    name               = "azure"
    client_id          = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    client_secret      = "shhh"
    scope              = "openid email"
    discovery_endpoint = "https://login.microsoftonline.com/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/.well-known/openid-configuration"
  }
}

resource "stackit_workflows_instance" "full" {
  project_id                  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region                      = "eu01"
  display_name                = "production-workflows"
  description                 = "Production STACKIT Workflows instance."
  version                     = "workflows-3.0-airflow-3.1"
  enable_stackit_example_dags = false
  enable_airflow_example_dags = false
  observability_id            = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

  network = {
    id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }

  identity_provider = {
    type               = "oauth2"
    name               = "azure"
    client_id          = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    client_secret      = "shhh"
    scope              = "openid email"
    discovery_endpoint = "https://login.microsoftonline.com/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/.well-known/openid-configuration"
    api_audience       = ["api://workflows"]
  }
}

# Only use the import statement, if you want to import an existing Workflows instance.
# The client_secret cannot be imported — it is never returned by the API.
import {
  to = stackit_workflows_instance.import-example
  id = "${var.project_id},${var.region},${var.workflows_instance_id}"
}
