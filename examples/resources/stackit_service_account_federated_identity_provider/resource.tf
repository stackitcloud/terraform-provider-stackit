resource "stackit_service_account" "sa" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "my-service-account"
}

resource "stackit_service_account_federated_identity_provider" "provider" {
  project_id            = stackit_service_account.sa.project_id
  service_account_email = stackit_service_account.sa.email
  name                  = "gh-actions"
  issuer                = "https://token.actions.githubusercontent.com"

  assertions = [
    {
      item     = "aud"
      operator = "equals"
      value    = "sts.accounts.stackit.cloud"
    }
    {
      item     = "sub"
      operator = "equals"
      value    = "repo:stackitcloud/terraform-provider-stackit:ref:refs/heads/main"
    }
  ]
}

# Only use the import statement, if you want to import an existing federated identity provider
import {
  to = stackit_service_account_federated_identity_provider.import-example
  id = "${var.project_id},${var.service_account_email},${var.federation_id}"
}
