
variable "project_id" {}

variable "alertgroup_name" {}
variable "alert_rule_name" {}
variable "alert_rule_expression" {}
variable "record_rule_name" {}

variable "instance_name" {}
variable "plan_name" {}
variable "grafana_admin_enabled" {}

variable "logalertgroup_name" {}
variable "logalertgroup_alert" {}
variable "logalertgroup_expression" {}


variable "scrapeconfig_name" {}
variable "scrapeconfig_metrics_path" {}
variable "scrapeconfig_targets_url" {}


resource "stackit_observability_alertgroup" "alertgroup" {
  project_id  = var.project_id
  instance_id = stackit_observability_instance.instance.instance_id
  name        = var.alertgroup_name
  rules = [
    {
      alert      = var.alert_rule_name
      expression = var.alert_rule_expression
    },
    {
      record     = var.record_rule_name
      expression = var.alert_rule_expression
    }
  ]
}

resource "stackit_observability_credential" "credential" {
  project_id  = var.project_id
  instance_id = stackit_observability_instance.instance.instance_id
}


resource "stackit_observability_instance" "instance" {
  project_id            = var.project_id
  name                  = var.instance_name
  plan_name             = var.plan_name
  grafana_admin_enabled = var.grafana_admin_enabled
}

resource "stackit_observability_logalertgroup" "logalertgroup" {
  project_id  = var.project_id
  instance_id = stackit_observability_instance.instance.instance_id
  name        = var.logalertgroup_name
  rules = [
    {
      alert      = var.logalertgroup_alert
      expression = var.logalertgroup_expression
    }
  ]
}


resource "stackit_observability_scrapeconfig" "scrapeconfig" {
  project_id   = var.project_id
  instance_id  = stackit_observability_instance.instance.instance_id
  name         = var.scrapeconfig_name
  metrics_path = var.scrapeconfig_metrics_path

  targets = [{ urls = [var.scrapeconfig_targets_url] }]
}




