variable "project_id" {}
variable "algorithm" {}
variable "display_name" {}
variable "protection" {}
variable "purpose" {}

resource "stackit_kms_key_ring" "key_ring" {
  project_id   = var.project_id
  display_name = var.display_name
}

resource "stackit_kms_key" "key" {
  algorithm = var.algorithm
  display_name = var.display_name
  key_ring_id = stackit_kms_key_ring.key_ring.key_ring_id
  project_id = var.project_id
  protection = var.protection
  purpose = var.purpose
}