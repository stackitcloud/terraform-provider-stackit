variable "project_id" {}
variable "waf_configuration_name" {}
variable "rule_set_name" {}
variable "type" {}
variable "waf_configuration_label" {}
variable "custom_rule_group_name" {}
variable "action" {}
variable "operator_type" {}
variable "variable_type" {}

resource "stackit_alb_waf_custom_rule_group" "custom_rule_group" {
  project_id = var.project_id
  name       = var.custom_rule_group_name
  rules = [
    {
      behavior = {
        action = var.action
      }
      conditions = [
        {
          operator = {
            type = var.operator_type
          }
          variable = {
            type = var.variable_type
          }
        }
      ]
    }
  ]
}

resource "stackit_alb_waf_managed_rule_set" "managed_rule_set" {
  project_id = var.project_id
  type       = var.type
  name       = var.rule_set_name
}
resource "stackit_alb_waf_configuration" "waf_instance" {
  project_id             = var.project_id
  name                   = var.waf_configuration_name
  managed_rule_set_name  = stackit_alb_waf_managed_rule_set.managed_rule_set.name
  custom_rule_group_name = stackit_alb_waf_custom_rule_group.custom_rule_group.name
  labels = {
    label1 = var.waf_configuration_label
  }
}

