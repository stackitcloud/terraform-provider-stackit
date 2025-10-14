variable "project_id" {}
variable "description" {}
variable "display_name" {}
variable "region" {}

resource "stackit_kms_key_ring" "key_ring" {
  project_id = var.project_id
  description = var.description
  display_name = var.display_name
  region = var.region
}

data "stackit_kms_key_ring" "key_ring" {
  project_id = var.project_id
  key_ring_id = stackit_kms_key_ring.key_ring.key_ring_id
}