resource "stackitprivatepreview_postgresflexalpha_instance" "ptlsdbsrv" {
  project_id      = var.project_id
  name            = "pgsql-example-instance"
  acl             = ["0.0.0.0/0"]
  backup_schedule = "0 0 * * *"
  flavor = {
    cpu = 2
    ram = 4
  }
  replicas = 3
  storage = {
    class = "premium-perf2-stackit"
    size  = 5
  }
  encryption = {
    #    key_id = stackit_kms_key.key.key_id
    #    keyring_id = stackit_kms_keyring.keyring.keyring_id
    key_id = var.key_id
    keyring_id = var.keyring_id
    key_version = var.key_version
    service_account = var.sa_email
  }
  network = {
    acl             = ["0.0.0.0/0", "193.148.160.0/19"]
    access_scope = "SNA"
  }
  version = 14
}


resource "stackitprivatepreview_sqlserverflexalpha_user" "ptlsdbuser" {
  project_id  = stackitprivatepreview_postgresflexalpha_instance.ptlsdbsrv.project_id
  instance_id = stackitprivatepreview_postgresflexalpha_instance.ptlsdbsrv.id
  username    = var.db_username
  roles       = ["createdb", "login", "createrole"]
}
