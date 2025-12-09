
resource "stackit_authorization_folder_role_assignment" "serviceaccount" {
  resource_id = stackit_resourcemanager_folder.test.folder_id
  role        = "owner"
  subject     = var.test_service_account
}