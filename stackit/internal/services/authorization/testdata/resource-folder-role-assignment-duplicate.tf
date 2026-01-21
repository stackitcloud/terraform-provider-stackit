variable "name" {}
variable "role" {}
variable "owner_email" {}
variable "subject" {}
variable "parent_container_id" {}

resource "stackit_resourcemanager_folder" "folder" {
  name                = var.name
  owner_email         = var.owner_email
  parent_container_id = var.parent_container_id
}

resource "stackit_authorization_folder_role_assignment" "fra" {
  resource_id = stackit_resourcemanager_folder.folder.folder_id
  role        = var.role
  subject     = var.subject
}

# Second assignment – duplicates stackit_authorization_folder_role_assignment.fra (same resource_id, role, subject)
resource "stackit_authorization_folder_role_assignment" "fra2" {
  resource_id = stackit_resourcemanager_folder.folder.folder_id
  role        = var.role
  subject     = var.subject
}