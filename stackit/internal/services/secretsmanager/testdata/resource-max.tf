variable "project_id" {}
variable "instance_name" {}
variable "user_description" {}
variable "write_enabled" {}
variable "acl1" {}
variable "acl2" {}
variable "service_account_mail" {}
variable "use_kms_key" {}

resource "stackit_secretsmanager_instance" "instance" {
  project_id = var.project_id
  name       = var.instance_name
  acls = [
    var.acl1,
    var.acl2,
  ]
}

resource "stackit_secretsmanager_user" "user" {
  project_id    = var.project_id
  instance_id   = stackit_secretsmanager_instance.instance.instance_id
  description   = var.user_description
  write_enabled = var.write_enabled
}


data "stackit_secretsmanager_instance" "instance" {
  project_id  = var.project_id
  instance_id = stackit_secretsmanager_instance.instance.instance_id
}

data "stackit_secretsmanager_user" "user" {
  project_id  = var.project_id
  instance_id = stackit_secretsmanager_instance.instance.instance_id
  user_id     = stackit_secretsmanager_user.user.user_id
}

# just needed for the test setup for secretsmanager with own keys
resource "stackit_kms_keyring" "keyring" {
  project_id   = var.project_id
  display_name = var.instance_name
}

# just needed for the test setup for secretsmanager with own keys
resource "stackit_kms_key" "key" {
  project_id   = var.project_id
  keyring_id   = stackit_kms_keyring.keyring.keyring_id
  display_name = var.instance_name
  protection   = "software"
  algorithm    = "aes_256_gcm"
  purpose      = "symmetric_encrypt_decrypt"
}

resource "stackit_secretsmanager_instance" "instance_with_key" {
  project_id = var.project_id
  name       = var.instance_name
  acls = [
    var.acl1,
    var.acl2,
  ]

  kms_key = var.use_kms_key ? {
    key_id                = stackit_kms_key.key.key_id
    key_ring_id           = stackit_kms_keyring.keyring.keyring_id
    key_version           = 1
    service_account_email = var.service_account_mail
  } : null
}


data "stackit_secretsmanager_instance" "instance_with_key" {
  project_id  = var.project_id
  instance_id = stackit_secretsmanager_instance.instance_with_key.instance_id
}
