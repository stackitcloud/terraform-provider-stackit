resource "stackit_kms_keyring" "keyring" {
  project_id   = var.project_id
  display_name = "msh-keyring01"
  description  = "This is a test keyring for private endpoints"
}

resource "stackit_kms_key" "key" {
  project_id   = var.project_id
  keyring_id   = stackit_kms_keyring.keyring.keyring_id
  display_name = "msh-key01"
  protection   = "software"
  algorithm    = "aes_256_gcm"
  purpose      = "symmetric_encrypt_decrypt"
  access_scope = "SNA"
}

output "keyid" {
  value = stackit_kms_key.key.key_id
}

# resource "stackitalpha_postgresflexalpha_instance" "ptlsdbsrv" {
#   project_id      = var.project_id
#   name            = "example-instance"
#   acl             = ["0.0.0.0/0"]
#   backup_schedule = "0 0 * * *"
#   flavor = {
#     cpu = 2
#     ram = 4
#   }
#   replicas = 1
#   storage = {
#     class = "premium-perf2-stackit"
#     size  = 5
#   }
#   version = 14
#   encryption = {
#     key_id = stackitalpha_kms_key.key.id
#     key_ring_id = stackitalpha_kms_keyring.keyring.keyring_id
#     key_version = "1"
#     service_account = var.sa_email
#   }
#   network = {
#     access_scope = "SNA"
#   }
# }

resource "stackitprivatepreview_sqlserverflexalpha_instance" "ptlsdbsqlsrv" {
  project_id      = var.project_id
  name            = "msh-example-instance-002"
  backup_schedule = "0 3 * * *"
  retention_days = 31
  flavor = {
    cpu = 4
    ram = 16
    node_type = "Single"
  }
  storage = {
    class = "premium-perf2-stackit"
    size  = 50
  }
  version = 2022
  encryption = {
    key_id = stackit_kms_key.key.key_id
    keyring_id = stackit_kms_keyring.keyring.keyring_id
#    key_id = var.key_id
#    keyring_id = var.keyring_id
    key_version = var.key_version
    service_account = var.sa_email
  }
  network = {
    acl             = ["0.0.0.0/0", "193.148.160.0/19"]
    access_scope = "SNA"
  }
}

# data "stackitalpha_sqlserverflexalpha_instance" "test" {
#   project_id = var.project_id
#   instance_id = var.instance_id
#   region = "eu01"
# }

# output "test" {
#   value = data.stackitalpha_sqlserverflexalpha_instance.test
# }

# data "stackitalpha_sqlserverflexalpha_user" "testuser" {
#   project_id = var.project_id
#   instance_id = var.instance_id
#   region = "eu01"
# }
