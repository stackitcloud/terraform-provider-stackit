variable "project_id" {}
variable "type" {}
variable "name" {}
variable "configured_groups" {
  type = map(object({
    rules = map(object({
      mode = string
    }))
  }))
}

resource "stackit_alb_waf_managed_rule_set" "managed_rule_set" {
  project_id        = var.project_id
  type              = var.type
  name              = var.name
  configured_groups = var.configured_groups
}

