variable "project_id" {}
variable "availability_zone" {}
variable "size" {}

resource "stackit_volume" "volume_size" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  size              = var.size
}

resource "stackit_volume" "volume_source" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  source = {
    id   = stackit_volume.volume_size.volume_id
    type = "volume"
  }
}