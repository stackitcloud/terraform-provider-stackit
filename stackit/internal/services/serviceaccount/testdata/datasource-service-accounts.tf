data "stackit_service_accounts" "list" {
  project_id = stackit_service_account.sa.project_id
}