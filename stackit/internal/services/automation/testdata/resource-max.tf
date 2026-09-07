variable "project_id" {}
variable "region" {}
variable "template_id" {}
variable "name" {}
variable "description" {}
variable "rrule" {}
variable "retention_count" {}

resource "stackit_volume_automation" "test" {
  project_id  = var.project_id
  region      = var.region
  template_id = var.template_id
  name        = var.name
  description = var.description
  input = {
    volume_recovery_point_management = {
      inherit_volume_labels = true
      recovery_point_labels = {
        "created-by" = "tf-acc-test"
      }
      snapshot_retention_policy = {
        kind  = "count"
        value = var.retention_count
      }
    }
  }
  triggers = {
    schedule = {
      rrule = var.rrule
    }
  }
}
