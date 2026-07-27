
variable "project_id" {}
variable "name" {}
variable "action" {}
variable "operator_type" {}
variable "variable_type" {}

resource "stackit_alb_waf_custom_rule_group" "custom_rule_group" {
  project_id = var.project_id
  name       = var.name
  rules = [
    {
      behaviour = {
        action = var.action
      }
      conditions = [
        {
          operator = {
            type  = var.operator_type
            value = "dummy"
          }
          variable = {
            type  = var.variable_type
          }
        }
      ]
    }
  ]
}
