
variable "project_id" {}
variable "name" {}
variable "max_message_size_kib" {}
variable "max_messages_per_hour" {}

resource "stackit_intake_runner" "example" {
  project_id            = var.project_id
  name                  = var.name
  max_message_size_kib  = var.max_message_size_kib
  max_messages_per_hour = var.max_messages_per_hour
}