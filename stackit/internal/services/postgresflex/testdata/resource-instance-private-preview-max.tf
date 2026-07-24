variable "project_id" {}
variable "kek_key_version" {}
variable "service_account_email" {}
variable "keyring_display_name" {}
variable "display_name" {}
variable "protection" {}
variable "algorithm" {}
variable "purpose" {}

variable "name" {}
variable "acl" {}
variable "access_scope" {}
variable "backup_schedule" {}
variable "flavor_id" {}
variable "storage_class" {}
variable "storage_size" {}
variable "instance_version" {}
variable "retention_days" {}
variable "flavor_cpu" {}
variable "flavor_ram" {}
variable "replicas" {}
variable "region" {}

# no test candidate, just needed for the testing setup
resource "stackit_kms_keyring" "keyring" {
  project_id   = var.project_id
  display_name = var.keyring_display_name
}

resource "stackit_kms_key" "key" {
  project_id   = var.project_id
  keyring_id   = stackit_kms_keyring.keyring.keyring_id
  protection   = var.protection
  algorithm    = var.algorithm
  display_name = var.display_name
  purpose      = var.purpose
}

# test candidate
resource "stackit_postgresflex_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  network = {
    acl          = [var.acl]
    access_scope = var.access_scope
  }
  backup_schedule = var.backup_schedule
  flavor_id       = var.flavor_id
  storage = {
    class = var.storage_class
    size  = var.storage_size
  }
  version        = var.instance_version
  retention_days = var.retention_days

  encryption = {
    kek_key_id      = stackit_kms_key.key.key_id
    kek_keyring_id  = stackit_kms_keyring.keyring.keyring_id
    kek_key_version = var.kek_key_version
    service_account = var.service_account_email
  }
}