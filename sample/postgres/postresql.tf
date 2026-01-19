data "stackitprivatepreview_postgresflexalpha_flavor" "pgsql_flavor" {
  project_id = var.project_id
  region = "eu01"
  cpu = 2
  ram = 4
  node_type = "Single"
  storage_class = "premium-perf2-stackit"
}

resource "stackitprivatepreview_postgresflexalpha_instance" "msh-sna-pe-example" {
  project_id      = var.project_id
  name            = "mshpetest2"
  backup_schedule = "0 0 * * *"
  retention_days  = 45
  flavor_id = data.stackitprivatepreview_postgresflexalpha_flavor.pgsql_flavor.flavor_id
  replicas = 1
  storage = {
    # class = "premium-perf2-stackit"
    performance_class = data.stackitprivatepreview_postgresflexalpha_flavor.pgsql_flavor.storage_class
    size  = 10
  }
  encryption = {
    #    key_id = stackit_kms_key.key.key_id
    #    keyring_id = stackit_kms_keyring.keyring.keyring_id
    kek_key_id          = var.key_id
    kek_key_ring_id      = var.keyring_id
    kek_key_version     = var.key_version
    service_account = var.sa_email
  }
  network = {
    acl          = ["0.0.0.0/0", "193.148.160.0/19", "170.85.2.177/32"]
    access_scope = "SNA"
  }
  version = 14
}

resource "stackitprivatepreview_postgresflexalpha_user" "ptlsdbadminuser" {
  project_id  = var.project_id
  instance_id = stackitprivatepreview_postgresflexalpha_instance.msh-sna-pe-example.instance_id
  username    = var.db_admin_username
  roles       = ["createdb", "login"]
  # roles     = ["createdb", "login", "createrole"]
}

resource "stackitprivatepreview_postgresflexalpha_user" "ptlsdbuser" {
  project_id  = var.project_id
  instance_id = stackitprivatepreview_postgresflexalpha_instance.msh-sna-pe-example.instance_id
  username    = var.db_username
  roles       = ["login"]
  # roles     = ["createdb", "login", "createrole"]
}

resource "stackitprivatepreview_postgresflexalpha_database" "example" {
  count = 25
  depends_on = [stackitprivatepreview_postgresflexalpha_user.ptlsdbadminuser]
  project_id  = var.project_id
  instance_id = stackitprivatepreview_postgresflexalpha_instance.msh-sna-pe-example.instance_id
  name        = "${var.db_name}${count.index}"
  owner       = var.db_admin_username
}

# data "stackitprivatepreview_postgresflexalpha_instance" "datapsql" {
#   project_id = var.project_id
#   instance_id = var.instance_id
#   region = "eu01"
# }

# output "psql_instance_id" {
#   value = data.stackitprivatepreview_postgresflexalpha_instance.datapsql.instance_id
# }

output "psql_user_password" {
  value = stackitprivatepreview_postgresflexalpha_user.ptlsdbuser.password
  sensitive = true
}

output "psql_user_conn" {
  value = stackitprivatepreview_postgresflexalpha_user.ptlsdbuser.connection_string
  sensitive = true
}
