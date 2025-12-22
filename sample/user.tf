data "stackitprivatepreview_postgresflexalpha_user" "example" {
  project_id  = stackitprivatepreview_postgresflexalpha_instance.ptlsdbsrv.project_id
  instance_id = stackitprivatepreview_postgresflexalpha_instance.ptlsdbsrv.id
  user_id     = 1
}

resource "stackitprivatepreview_postgresflexalpha_user" "ptlsdbuser" {
  project_id  = stackitprivatepreview_postgresflexalpha_instance.ptlsdbsrv.project_id
  instance_id = stackitprivatepreview_postgresflexalpha_instance.ptlsdbsrv.id
  username    = var.db_username
  # roles       = ["createdb", "login", "createrole"]
  roles       = ["createdb", "login"]
}

resource "stackitprivatepreview_sqlserverflexalpha_user" "ptlsdbuser" {
  project_id  = stackitprivatepreview_sqlserverflexalpha_instance.ptlsdbsqlsrv.project_id
  instance_id = stackitprivatepreview_sqlserverflexalpha_instance.ptlsdbsqlsrv.id
  username    = var.db_username
  roles       = ["login"]
}
