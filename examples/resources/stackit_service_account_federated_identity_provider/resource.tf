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
    },
    {
      item     = "sub"
      operator = "equals"
      value    = "repo:stackitcloud/terraform-provider-stackit:ref:refs/heads/main"
    }
  ]
}
