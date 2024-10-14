resource "stackit_server_service_account_attach" "attached_service_account" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  service_account_email = "service-account@stackit.cloud"
}
