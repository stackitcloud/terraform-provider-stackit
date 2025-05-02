variable "project_id" {}

variable "name" {}
variable "security_group_rule_direction" {}

resource "stackit_security_group" "security_group" {
  project_id = var.project_id
  name       = var.name
}

resource "stackit_security_group_rule" "security_group_rule" {
  project_id        = var.project_id
  security_group_id = stackit_security_group.security_group.security_group_id
  direction         = var.security_group_rule_direction
}