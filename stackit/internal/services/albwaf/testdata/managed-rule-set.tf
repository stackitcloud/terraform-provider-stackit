
variable "project_id" {}
variable "type" {}
variable "name" {}

resource "stackit_alb_waf_managed_rule_set" "managed_rule_set" {
  project_id = var.project_id
  type       = var.type
  name       = var.name
}
