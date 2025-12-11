resource "stackitalpha_kms_keyring" "keyring" {
  project_id   = var.project_id
  display_name = "keyring01"
  description  = "This is a test keyring for private endpoints"
}

resource "stackitalpha_kms_key" "key" {
  project_id   = var.project_id
  keyring_id   = stackitalpha_kms_keyring.keyring.keyring_id
  display_name = "key01"
  protection   = "software"
  algorithm    = "aes_256_gcm"
  purpose      = "symmetric_encrypt_decrypt"
  access_scope = "SNA"
}

resource "stackitalpha_postgresflexalpha_instance" "ptlsdbsrv" {
  project_id      = var.project_id
  name            = "example-instance"
  acl             = ["0.0.0.0/0"]
  backup_schedule = "0 0 * * *"
  flavor = {
    cpu = 2
    ram = 4
  }
  replicas = 3
  storage = {
    class = "premium-perf12-stackit"
    size  = 5
  }
  version = 14
  encryption = {
    key_id = stackitalpha_kms_key.key.id
    key_ring_id = stackitalpha_kms_keyring.keyring.keyring_id
    key_version = "1"
    service_account = var.sa_email
  }
  network = {
    access_scope = "SNA"
  }
}
