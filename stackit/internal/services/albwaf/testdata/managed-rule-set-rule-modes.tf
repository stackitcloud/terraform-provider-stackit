variable "project_id" {}
variable "type" {}
variable "name" {}
variable "rule_modes" {
  type = map(string)
}

resource "stackit_alb_waf_managed_rule_set" "managed_rule_set" {
  project_id = var.project_id
  type       = var.type
  name       = var.name
  rule_modes = var.rule_modes
}
