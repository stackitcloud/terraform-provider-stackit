variable "project_id" {}
variable "waf_name" {}
variable "rule_set_name" {}
variable "type" {}
variable "waf_label" {}

resource "stackit_alb_waf_managed_rule_set" "managed_rule_set" {
  project_id = var.project_id
  type       = var.type
  name       = var.rule_set_name
}
resource "stackit_alb_waf" "waf_instance" {
  project_id            = var.project_id
  name                  = var.waf_name
  managed_rule_set_name = stackit_alb_waf_managed_rule_set.managed_rule_set.name
  labels = {
    label1 = var.waf_label
  }
}

