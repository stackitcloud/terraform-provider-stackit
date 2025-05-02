variable "project_id" {}

variable "name" {}
variable "description" {}
variable "description_rule" {}
variable "label" {}
variable "stateful" {}
variable "direction" {}
variable "ether_type" {}
variable "ip_range" {}
variable "port" {}
variable "protocol" {}
variable "icmp_code" {}
variable "icmp_type" {}
variable "name_remote" {}

resource "stackit_security_group" "security_group" {
  project_id  = var.project_id
  name        = var.name
  description = var.description
  labels = {
    "acc-test" : var.label
  }
  stateful = var.stateful
}

resource "stackit_security_group_rule" "security_group_rule" {
  project_id        = var.project_id
  security_group_id = stackit_security_group.security_group.security_group_id
  direction         = var.direction

  description = var.description_rule
  ether_type  = var.ether_type
  port_range = {
    min = var.port
    max = var.port
  }
  protocol = {
    name = var.protocol
  }
  ip_range = var.ip_range
}

resource "stackit_security_group_rule" "security_group_rule_icmp" {
  project_id        = var.project_id
  security_group_id = stackit_security_group.security_group.security_group_id
  direction         = var.direction

  description = var.description_rule
  ether_type  = var.ether_type
  icmp_parameters = {
    code = var.icmp_code
    type = var.icmp_type
  }
  protocol = {
    name = "icmp"
  }
  ip_range = var.ip_range
}

resource "stackit_security_group" "security_group_remote" {
  project_id  = var.project_id
  name        = var.name_remote
}

resource "stackit_security_group_rule" "security_group_rule_remote_security_group" {
  project_id        = var.project_id
  security_group_id = stackit_security_group.security_group.security_group_id
  direction         = var.direction

  remote_security_group_id = stackit_security_group.security_group_remote.security_group_id
}