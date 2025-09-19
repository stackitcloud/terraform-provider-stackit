variable "project_id" {}
variable "name" {}
variable "db_version" {}
variable "plan_name" {}

variable "parameters_sgw_acl" {}
variable "parameters_consumer_timeout" {}
variable "parameters_enable_monitoring" {}
variable "parameters_graphite" {}
variable "parameters_max_disk_threshold" {}
variable "parameters_metrics_frequency" {}
variable "parameters_metrics_prefix" {}
variable "parameters_plugins" {}
variable "parameters_roles" {}
variable "parameters_syslog" {}

resource "stackit_rabbitmq_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  version    = var.db_version
  plan_name  = var.plan_name
  parameters = {
    sgw_acl                = var.parameters_sgw_acl
    consumer_timeout        = var.parameters_consumer_timeout
    enable_monitoring       = var.parameters_enable_monitoring
    graphite                = var.parameters_graphite
    max_disk_threshold      = var.parameters_max_disk_threshold
    metrics_frequency       = var.parameters_metrics_frequency
    metrics_prefix          = var.parameters_metrics_prefix
    plugins                 = var.parameters_plugins
    roles                   = var.parameters_roles
    syslog                  = var.parameters_syslog
  }
}

resource "stackit_rabbitmq_credential" "credential" {
  project_id  = var.project_id
  instance_id = stackit_rabbitmq_instance.instance.instance_id
}
