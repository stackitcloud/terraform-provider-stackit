variable "project_id" {
  type = string
}

variable "source_ip" {
  type = string
}

resource "stackit_edgecloud_instance" "target" {
  project_id   = var.project_id
  display_name = "edge"
  plan_id      = "4916c0e2-e719-445a-9920-58e491cd06c5"
  description  = "cats live on the edge"
  region       = "eu01"
}

resource "stackit_ufw_instance" "example" {
  project_id  = var.project_id
  instance_id = stackit_edgecloud_instance.target.instance_id
  product     = "edge-cloud"
  source_ip   = var.source_ip
  type        = "ACL"
}