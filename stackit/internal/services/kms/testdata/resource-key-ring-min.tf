variable "project_id" {}

variable "display_name" {}

resource "stackit_kms_key_ring" "key_ring" {
  project_id   = var.project_id
  display_name = var.display_name
}

data "stackit_kms_key_ring" "key_ring" {
  project_id = var.project_id
  key_ring_id = stackit_kms_key_ring.key_ring.key_ring_id
}