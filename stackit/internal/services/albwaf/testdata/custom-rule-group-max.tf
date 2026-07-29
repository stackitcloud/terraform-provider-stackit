
variable "project_id" {}
variable "name" {}
variable "description" {}
variable "action" {}
variable "log" {}
variable "log_msg" {}
variable "operator_type" {}
variable "operator_value" {}
variable "transformation" {}
variable "variable_type" {}
variable "variable_value" {}

resource "stackit_alb_waf_custom_rule_group" "custom_rule_group" {
  project_id = var.project_id
  name       = var.name
  rules = [
    {
      description = var.description
      behavior = {
        action = var.action
        log    = var.log
        logMsg = var.log_msg
      }
      conditions = [
        {
          operator = {
            type  = var.operator_type
            value = var.operator_value
          }
          transformations = [
            var.transformation
          ]
          variable = {
            type  = var.variable_type
            value = var.variable_value
          }
        }
      ]
    }
  ]
}
