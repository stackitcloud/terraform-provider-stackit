variable "project_id" {}
variable "name" {}
variable "plan_name" {}
variable "logme_version" {}
variable "params_enable_monitoring" {}
variable "params_fluentd_tcp" {}
variable "params_fluentd_tls" {}
variable "params_fluentd_tls_ciphers" {}
variable "params_fluentd_tls_max_version" {}
variable "params_fluentd_tls_min_version" {}
variable "params_fluentd_tls_version" {}
variable "params_fluentd_udp" {}
variable "params_graphite" {}
variable "params_ism_deletion_after" {}
variable "params_ism_jitter" {}
variable "params_ism_job_interval" {}
variable "params_java_heapspace" {}
variable "params_java_maxmetaspace" {}
variable "params_max_disk_threshold" {}
variable "params_metrics_frequency" {}
variable "params_metrics_prefix" {}
variable "params_monitoring_instance_id" {}
variable "params_opensearch_tls_cipher1" {}
variable "params_opensearch_tls_cipher2" {}
variable "params_opensearch_tls_protocol1" {}
variable "params_opensearch_tls_protocol2" {}
variable "params_sgw_acl" {}
variable "params_syslog1" {}
variable "params_syslog2" {}

resource "stackit_logme_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  plan_name  = var.plan_name
  version    = var.logme_version

  parameters = {
    enable_monitoring        = var.params_enable_monitoring
    fluentd_tcp              = var.params_fluentd_tcp
    fluentd_tls              = var.params_fluentd_tls
    fluentd_tls_ciphers      = var.params_fluentd_tls_ciphers
    fluentd_tls_max_version  = var.params_fluentd_tls_max_version
    fluentd_tls_min_version  = var.params_fluentd_tls_min_version
    fluentd_tls_version      = var.params_fluentd_tls_version
    fluentd_udp              = var.params_fluentd_udp
    graphite                 = var.params_graphite
    ism_deletion_after       = var.params_ism_deletion_after
    ism_jitter               = var.params_ism_jitter
    ism_job_interval         = var.params_ism_job_interval
    java_heapspace           = var.params_java_heapspace
    java_maxmetaspace        = var.params_java_maxmetaspace
    max_disk_threshold       = var.params_max_disk_threshold
    metrics_frequency        = var.params_metrics_frequency
    metrics_prefix           = var.params_metrics_prefix
    opensearch_tls_ciphers   = [var.params_opensearch_tls_cipher1, var.params_opensearch_tls_cipher2]
    opensearch_tls_protocols = [var.params_opensearch_tls_protocol1, var.params_opensearch_tls_protocol2]
    sgw_acl                  = var.params_sgw_acl
    syslog                   = [var.params_syslog1, var.params_syslog2]

  }
}

resource "stackit_logme_credential" "credential" {
  project_id  = stackit_logme_instance.instance.project_id
  instance_id = stackit_logme_instance.instance.instance_id
}


data "stackit_logme_instance" "instance" {
  project_id  = stackit_logme_instance.instance.project_id
  instance_id = stackit_logme_instance.instance.instance_id
}

data "stackit_logme_credential" "credential" {
  project_id    = stackit_logme_credential.credential.project_id
  instance_id   = stackit_logme_credential.credential.instance_id
  credential_id = stackit_logme_credential.credential.credential_id
}
