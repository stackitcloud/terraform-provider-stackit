variable "project_id" {}
variable "template_id" {}
variable "name" {}
variable "rrule" {}
variable "input" {}

resource "stackit_volume_automation" "test" {
  project_id  = var.project_id
  template_id = var.template_id
  name        = var.name
  input       = var.input
  triggers = {
    schedule = {
      rrule = var.rrule
    }
  }
}
