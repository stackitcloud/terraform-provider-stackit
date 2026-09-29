variable "project_id" {}
variable "name" {}
variable "file_content" {}
variable "file_content_version" {}
variable "label_key" {}
variable "label_value" {}

resource "stackit_lb_ip_list" "iplist" {
  project_id           = var.project_id
  name                 = var.name
  file_content         = var.file_content
  file_content_version = var.file_content_version
  labels = {
    (var.label_key) = var.label_value
  }
}
