variable "project_id" {}
variable "name" {}
variable "acl1" {}
variable "flavor_id" {}
variable "storage_class" {}
variable "storage_size" {}
variable "access_scope" {}
variable "retention_days" {}
variable "backup_schedule" {}
variable "username" {}
variable "role" {}
variable "server_version" {}
variable "region" {}

variable "kek_key_version" {}
variable "service_account_email" {}

variable "keyring_display_name" {}
variable "display_name" {}
variable "protection" {}
variable "algorithm" {}
variable "purpose" {}


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

resource "stackit_sqlserverflex_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  flavor_id  = var.flavor_id
  storage = {
    class = var.storage_class
    size  = var.storage_size
  }
  network = {
    acl          = [var.acl1]
    access_scope = var.access_scope
  }
  retention_days = var.retention_days
  version        = var.server_version
  encryption = {
    kek_key_id      = stackit_kms_key.key.key_id
    kek_keyring_id  = stackit_kms_keyring.keyring.keyring_id
    kek_key_version = var.kek_key_version
    service_account = var.service_account_email
  }
  backup_schedule = var.backup_schedule
  region          = var.region
}

resource "stackit_sqlserverflex_user" "user" {
  project_id  = stackit_sqlserverflex_instance.instance.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
  username    = var.username
  roles       = [var.role]
}

data "stackit_sqlserverflex_instance" "instance" {
  project_id  = var.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
}

data "stackit_sqlserverflex_user" "user" {
  project_id  = var.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
  user_id     = stackit_sqlserverflex_user.user.user_id
}
