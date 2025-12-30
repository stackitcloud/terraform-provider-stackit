resource "stackitprivatepreview_postgresflexalpha_instance" "ptlsdbsrv" {
  project_id      = var.project_id
  name            = "pgsql-example-instance"
  backup_schedule = "0 0 * * *"
  retention_days = 33
  flavor = {
    cpu = 2
    ram = 4
  }
  replicas = 1
  storage = {
    class = "premium-perf2-stackit"
    size  = 5
  }
  encryption = {
    #    key_id = stackit_kms_key.key.key_id
    #    keyring_id = stackit_kms_keyring.keyring.keyring_id
    key_id          = var.key_id
    keyring_id      = var.keyring_id
    key_version     = var.key_version
    service_account = var.sa_email
  }
  network = {
    acl          = ["0.0.0.0/0", "193.148.160.0/19"]
    access_scope = "SNA"
  }
  version = 14
}

# data "stackitprivatepreview_postgresflexalpha_instance" "datapsql" {
#   project_id = var.project_id
#   instance_id = "fdb6573e-2dea-4e1d-a638-9157cf90c3ba"
#   region = "eu01"
# }
#
# output "sample_psqlinstance" {
#   value = data.stackitprivatepreview_postgresflexalpha_instance.datapsql
# }
