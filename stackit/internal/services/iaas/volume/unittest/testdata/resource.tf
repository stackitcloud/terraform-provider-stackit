provider "stackit" {
  service_account_token = "mock-server-needs-no-auth"
}

variable "project_id" {}
variable "availability_zone" {}
variable "size" {}

resource "stackit_volume" "volume" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  size              = var.size
}