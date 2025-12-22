data "stackit_postgresflexalpha_user" "example" {
  project_id  = stackitalpha_postgresflexalpha_instance.ptlsdbsrv.project_id
  instance_id = stackitalpha_postgresflexalpha_instance.ptlsdbsrv.id
  user_id     = 1
}

resource "stackit_sqlserverflexalpha_user" "ptlsdbuser" {
  project_id  = stackitalpha_postgresflexalpha_instance.ptlsdbsrv.project_id
  instance_id = stackitalpha_postgresflexalpha_instance.ptlsdbsrv.id
  username    = var.db_username
  roles       = ["createdb", "login", "createrole"]
}