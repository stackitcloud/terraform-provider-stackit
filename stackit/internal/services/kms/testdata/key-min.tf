variable "project_id" {}

variable "keyring_display_name" {}
variable "display_name" {}
variable "protection" {}
variable "algorithm" {}
variable "purpose" {}

resource "stackit_kms_keyring" "keyring" {
  project_id   = var.project_id
  display_name = var.keyring_display_name
}

resource "stackit_kms_key" "key" {
  project_id   = var.project_id
  keyring_id   = stackit_kms_keyring.keyring.keyring_id
  protection   = var.protection
  algorithm    = var.algorithm
  display_name = var.display_name
  purpose      = var.purpose
}
