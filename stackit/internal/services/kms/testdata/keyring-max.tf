variable "project_id" {}

variable "display_name" {}
variable "description" {}

resource "stackit_kms_keyring" "keyring" {
  project_id   = var.project_id
  display_name = var.display_name
  description  = var.description
}
