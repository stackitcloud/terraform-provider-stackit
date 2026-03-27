
variable "folder_name" {}
variable "owner_email" {}
variable "parent_container_id" {}
variable "role_name" {}
variable "role_description" {}
variable "role_permissions_0" {}

resource "stackit_resourcemanager_folder" "folder" {
  name                = var.folder_name
  owner_email         = var.owner_email
  parent_container_id = var.parent_container_id
}

resource "stackit_authorization_folder_custom_role" "folder_custom_role" {
  resource_id = stackit_resourcemanager_folder.folder.folder_id
  name        = var.role_name
  description = var.role_description
  permissions = [var.role_permissions_0]
}
