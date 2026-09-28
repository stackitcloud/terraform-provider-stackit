variable "project_id" {}
variable "region" {}
variable "template_id" {}
variable "name" {}
variable "description" {}
variable "rrule" {}
variable "input" {}

resource "stackit_volume_automation" "test" {
  project_id  = var.project_id
  region      = var.region
  template_id = var.template_id
  name        = var.name
  description = var.description
  input       = var.input
  triggers = {
    schedule = {
      rrule = var.rrule
    }
  }
}
