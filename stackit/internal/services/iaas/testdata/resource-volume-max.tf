variable "project_id" {}
variable "availability_zone" {}
variable "name" {}
variable "size" {}
variable "description" {}
variable "performance_class" {}
variable "label" {}
variable "key_payload_base64" {}
variable "service_account_mail" {}

resource "stackit_volume" "volume_size" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  name              = var.name
  size              = var.size
  description       = var.description
  performance_class = var.performance_class
  labels = {
    "acc-test" : var.label
  }
}

resource "stackit_volume" "volume_source" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  name              = var.name
  description       = var.description
  # TODO: keep commented until IaaS API bug is resolved
  #performance_class = var.performance_class
  size = var.size
  source = {
    id   = stackit_volume.volume_size.volume_id
    type = "volume"
  }
  labels = {
    "acc-test" : var.label
  }
}

# just needed for the test setup for encrypted volumes
resource "stackit_kms_keyring" "keyring" {
  project_id   = var.project_id
  display_name = var.name
}

# just needed for the test setup for encrypted volumes
resource "stackit_kms_key" "key" {
  project_id   = var.project_id
  keyring_id   = stackit_kms_keyring.keyring.keyring_id
  display_name = var.name
  protection   = "software"
  algorithm    = "aes_256_gcm"
  purpose      = "symmetric_encrypt_decrypt"
}

resource "stackit_volume" "volume_encrypted_no_key_payload" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  name              = var.name
  size              = var.size
  description       = var.description
  performance_class = var.performance_class
  labels = {
    "acc-test" : var.label
  }

  encryption_parameters = {
    kek_key_id      = stackit_kms_key.key.key_id
    kek_key_version = 1
    kek_keyring_id  = stackit_kms_keyring.keyring.keyring_id
    service_account = var.service_account_mail
  }
}

resource "stackit_volume" "volume_encrypted_with_key_payload" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  name              = var.name
  size              = var.size
  description       = var.description
  performance_class = var.performance_class
  labels = {
    "acc-test" : var.label
  }

  encryption_parameters = {
    kek_key_id         = stackit_kms_key.key.key_id
    kek_key_version    = 1
    kek_keyring_id     = stackit_kms_keyring.keyring.keyring_id
    key_payload_base64 = var.key_payload_base64
    service_account    = var.service_account_mail
  }
}
