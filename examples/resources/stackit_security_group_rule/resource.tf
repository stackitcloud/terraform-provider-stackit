resource "stackit_security_group_rule" "example" {
  project_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  security_group_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  direction         = "ingress"
  icmp_parameters = {
    code = 0
    type = 8
  }
  protocol = {
    name = "icmp"
  }
}

# Only use the import statement, if you want to import an existing security group rule
# Note: There will be a conflict which needs to be resolved manually.
# Attribute "protocol.number" cannot be specified when "protocol.name" is specified.
import {
  to = stackit_security_group_rule.import-example
  id = "${var.project_id},${var.security_group_id},${var.security_group_rule_id}"
}