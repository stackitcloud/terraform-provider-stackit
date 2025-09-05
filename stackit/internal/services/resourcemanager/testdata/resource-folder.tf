
variable "parent_container_id" {}
variable "name" {}
variable "labels" {}
variable "owner_email" {}

resource "stackit_resourcemanager_folder" "example" {
  parent_container_id = var.parent_container_id
  name                = var.name
  labels              = var.labels
  owner_email         = var.owner_email
}
