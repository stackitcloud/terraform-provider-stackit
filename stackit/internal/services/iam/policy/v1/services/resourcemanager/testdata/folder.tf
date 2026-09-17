variable "organization_id" {}
variable "folder_name" {}
variable "folder_owner_email" {}
variable "role_bindings" {
  type = list(object({
    role    = string
    subject = string
  }))
}

resource "stackit_resourcemanager_folder" "folder" {
  name                = var.folder_name
  owner_email         = var.folder_owner_email
  parent_container_id = var.organization_id
}

resource "stackit_resourcemanager_folder_iam_policy_v1" "iam_policy" {
  resource_id   = stackit_resourcemanager_folder.folder.folder_id
  role_bindings = var.role_bindings
}