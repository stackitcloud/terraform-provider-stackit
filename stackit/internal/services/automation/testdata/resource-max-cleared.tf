variable "project_id" {}
variable "region" {}
variable "template_id" {}
variable "rrule" {}
variable "input" {}

resource "stackit_volume_automation" "test" {
  project_id  = var.project_id
  region      = var.region
  template_id = var.template_id
  input       = var.input
  triggers = {
    schedule = {
      rrule = var.rrule
    }
  }
}
