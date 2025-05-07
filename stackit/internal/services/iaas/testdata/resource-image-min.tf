variable "project_id" {}
variable "name" {}
variable "disk_format" {}
variable "local_file_path" {}

resource "stackit_image" "image" {
  project_id      = var.project_id
  name            = var.name
  disk_format     = var.disk_format
  local_file_path = var.local_file_path
}