variable "project_id" {}
variable "name" {}
variable "db_version" {}
variable "plan_name" {}
variable "observability_instance_plan_name" {}
variable "parameters_enable_monitoring" {}
variable "parameters_graphite" {}
variable "parameters_max_disk_threshold" {}
variable "parameters_metrics_frequency" {}
variable "parameters_metrics_prefix" {}
variable "parameters_sgw_acl" {}
variable "parameters_syslog" {}

resource "stackit_observability_instance" "observability_instance" {
  project_id = var.project_id
  name       = var.name
  plan_name  = var.observability_instance_plan_name
}

resource "stackit_mariadb_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  version    = var.db_version
  plan_name  = var.plan_name
  parameters = {
    enable_monitoring      = var.parameters_enable_monitoring
    graphite               = var.parameters_graphite
    max_disk_threshold     = var.parameters_max_disk_threshold
    metrics_frequency      = var.parameters_metrics_frequency
    metrics_prefix         = var.parameters_metrics_prefix
    monitoring_instance_id = stackit_observability_instance.observability_instance.instance_id
    sgw_acl                = var.parameters_sgw_acl
    syslog                 = [var.parameters_syslog]
  }
}

resource "stackit_mariadb_credential" "credential" {
  project_id  = var.project_id
  instance_id = stackit_mariadb_instance.instance.instance_id
}