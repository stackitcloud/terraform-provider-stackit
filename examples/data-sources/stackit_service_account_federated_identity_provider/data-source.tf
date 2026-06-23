data "stackit_service_account" "sa" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  email      = "sa01-8565oq1@sa.stackit.cloud"
}

data "stackit_service_account_federated_identity_provider" "provider" {
  project_id            = data.stackit_service_account.sa.project_id
  service_account_email = data.stackit_service_account.sa.email
  federation_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

