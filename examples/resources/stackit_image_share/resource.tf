resource "stackit_image" "example_image" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example-image"
  disk_format     = "qcow2"
  local_file_path = "./path/to/image.qcow2"
  min_disk_size   = 10
  min_ram         = 5
}


resource "stackit_image_share" "stackit_image_share" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  image_id   = stackit_image.image.image_id
  # either share to parent_organization or individual projects within the projects
  parent_organization = true
  /*projects = [
    "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  ]*/
}

import {
  to = stackit_image_share.stackit_image_share
  id = "${var.project_id},${var.region},${var.image_id}"
}