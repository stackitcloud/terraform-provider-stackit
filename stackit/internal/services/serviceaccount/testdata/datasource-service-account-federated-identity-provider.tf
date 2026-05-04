data "stackit_service_account_federated_identity_provider" "provider" {
  project_id            = stackit_service_account.sa.project_id
  service_account_email = stackit_service_account.sa.email
  federation_id         = stackit_service_account_federated_identity_provider.provider.federation_id
}
