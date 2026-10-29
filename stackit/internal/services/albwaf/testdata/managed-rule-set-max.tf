variable "project_id" {}
variable "type" {}
variable "name" {}
variable "groups" {
  type = map(object({
    rules = map(object({
      mode = string
    }))
  }))
}

resource "stackit_alb_waf_managed_rule_set" "managed_rule_set" {
  project_id = var.project_id
  type       = var.type
  name       = var.name
  groups     = var.groups
}

# Full inventory, to check the modes of rules the resource no longer manages
data "stackit_alb_waf_managed_rule_set" "managed_rule_set" {
  project_id = stackit_alb_waf_managed_rule_set.managed_rule_set.project_id
  name       = stackit_alb_waf_managed_rule_set.managed_rule_set.name
}
