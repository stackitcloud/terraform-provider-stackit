variable "project_id" {}
variable "name" {}
variable "name2" {}
variable "disk_format" {}
variable "local_file_path" {}
variable "url" {}

resource "stackit_image" "image" {
  project_id  = var.project_id
  name        = var.name
  disk_format = var.disk_format
  image_file = {
    local = {
      file_path = var.local_file_path
    }
  }
}

resource "stackit_image" "image2" {
  project_id  = var.project_id
  name        = var.name2
  disk_format = var.disk_format
  image_file = {
    download = {
      url = var.url
    }
  }
}