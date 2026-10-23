
variable "project_id" {}
variable "region" {}
variable "name" {}
variable "first_rule_description" {}
variable "first_rule_ip_acl" {}
variable "first_rule_set_uuid" {}
variable "second_rule_ip_acl" {}
variable "second_rule_read_only" {}
variable "second_rule_super_user" {}
variable "label" {}

resource "stackit_sfs_export_policy" "exportpolicy" {
  project_id = var.project_id
  region     = var.region
  name       = var.name
  rules = [{
    order       = 1
    description = var.first_rule_description
    ip_acl      = var.first_rule_ip_acl
    set_uuid    = var.first_rule_set_uuid
    }, {
    order      = 2
    ip_acl     = var.second_rule_ip_acl
    read_only  = var.second_rule_read_only
    super_user = var.second_rule_super_user
  }]
  labels = {
    label = var.label
  }
}
