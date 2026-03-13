data "stackit_service_account" "sa" {
  project_id = stackit_service_account.sa.project_id
  email      = stackit_service_account.sa.email
}