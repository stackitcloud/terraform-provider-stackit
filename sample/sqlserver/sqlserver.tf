# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: Apache-2.0

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

resource "stackitprivatepreview_sqlserverflexalpha_instance" "sqlsrv" {
  project_id      = var.project_id
  name            = "msh-example-instance-002"
  backup_schedule = "0 3 * * *"
  retention_days  = 31
  flavor_id       = data.stackitprivatepreview_sqlserverflexalpha_flavor.sqlserver_flavor.flavor_id
  storage = {
    class = "premium-perf2-stackit"
    size  = 50
  }
  version = 2022
  encryption = {
    key_id     = stackit_kms_key.key.key_id
    keyring_id = stackit_kms_keyring.keyring.keyring_id
    #key_id          = var.key_id
    #keyring_id      = var.keyring_id
    #key_version     = var.key_version
    key_version     = 1
    service_account = var.sa_email
  }
  network = {
    acl          = ["0.0.0.0/0", "193.148.160.0/19"]
    access_scope = "SNA"
  }
}

# data "stackitprivatepreview_sqlserverflexalpha_instance" "test" {
#   project_id = var.project_id
#   instance_id = var.instance_id
#   region = "eu01"
# }

# output "test" {
#   value = data.stackitprivatepreview_sqlserverflexalpha_instance.test
# }

resource "stackitprivatepreview_sqlserverflexalpha_user" "ptlsdbadminuser" {
  project_id  = var.project_id
  instance_id = stackitprivatepreview_sqlserverflexalpha_instance.sqlsrv.instance_id
  username    = var.db_admin_username
  roles       = ["##STACKIT_LoginManager##", "##STACKIT_DatabaseManager##"]
}

resource "stackitprivatepreview_sqlserverflexalpha_user" "ptlsdbuser" {
  project_id  = var.project_id
  instance_id = stackitprivatepreview_sqlserverflexalpha_instance.sqlsrv.instance_id
  username    = var.db_username
  roles       = ["##STACKIT_LoginManager##"]
}

