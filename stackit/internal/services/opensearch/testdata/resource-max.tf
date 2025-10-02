variable "project_id" {}
variable "name" {}
variable "instance_version" {}
variable "plan_name" {}
variable "enable_monitoring" {}
variable "graphite" {}
variable "java_garbage_collector" {}
variable "java_heapspace" {}
variable "java_maxmetaspace" {}
variable "max_disk_threshold" {}
variable "metrics_frequency" {}
variable "metrics_prefix" {}
variable "plugin" {}
variable "sgw_acl" {}
variable "syslog" {}
variable "tls_ciphers" {}
variable "tls_protocols" {}

variable "observability_instance_plan_name" {}

resource "stackit_opensearch_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  version    = var.instance_version
  plan_name  = var.plan_name
  parameters = {
    enable_monitoring = var.enable_monitoring
    graphite          = var.graphite
    java_garbage_collector = var.java_garbage_collector
    java_heapspace = var.java_heapspace
    java_maxmetaspace = var.java_maxmetaspace
    max_disk_threshold = var.max_disk_threshold
    metrics_frequency = var.metrics_frequency
    metrics_prefix = var.metrics_prefix
    monitoring_instance_id = stackit_observability_instance.instance.instance_id
    plugins = var.plugin != "" ? [var.plugin] : []
    sgw_acl = var.sgw_acl
    syslog = [var.syslog]
    tls_ciphers = [var.tls_ciphers]
    tls_protocols = [var.tls_protocols]
  }
}

resource "stackit_opensearch_credential" "credential" {
  project_id  = var.project_id
  instance_id = stackit_opensearch_instance.instance.instance_id
}

resource "stackit_observability_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  plan_name  = var.observability_instance_plan_name
}