variable "project_id" {}
variable "template_id" {}
variable "name" {}
variable "rrule" {}

resource "stackit_volume_automation" "test" {
  project_id  = var.project_id
  template_id = var.template_id
  name        = var.name
  input = {
    volume_recovery_point_management = {
      snapshot_retention_policy = {
        kind = "indefinitely"
      }
    }
  }
  triggers = {
    schedule = {
      rrule = var.rrule
    }
  }
}
