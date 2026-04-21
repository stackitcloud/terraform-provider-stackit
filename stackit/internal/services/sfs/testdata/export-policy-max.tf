
variable "project_id" {}
variable "region" {}
variable "name" {}
variable "first_rule_description" {}
variable "first_rule_ip_acl_1" {}
variable "first_rule_ip_acl_2" {}
variable "first_rule_set_uuid" {}
variable "second_rule_ip_acl_1" {}
variable "second_rule_ip_acl_2" {}
variable "second_rule_read_only" {}
variable "second_rule_super_user" {}

resource "stackit_sfs_export_policy" "exportpolicy" {
  project_id = var.project_id
  region     = var.region
  name       = var.name
  rules = [{
    order       = 1
    description = var.first_rule_description
    ip_acl = [
      var.first_rule_ip_acl_1,
      var.first_rule_ip_acl_2
    ]
    set_uuid = var.first_rule_set_uuid
    }, {
    order = 2
    ip_acl = [
      var.second_rule_ip_acl_1,
      var.second_rule_ip_acl_2
    ]
    read_only  = var.second_rule_read_only
    super_user = var.second_rule_super_user
  }]
}
