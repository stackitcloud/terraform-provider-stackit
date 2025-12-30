# data "stackitprivatepreview_postgresflexalpha_user" "example" {
#   project_id  = stackitprivatepreview_postgresflexalpha_instance.ptlsdbsrv.project_id
#   instance_id = stackitprivatepreview_postgresflexalpha_instance.ptlsdbsrv.instance_id
#   user_id     = 1
# }
#
# resource "stackitprivatepreview_postgresflexalpha_user" "ptlsdbuser" {
#   project_id  = stackitprivatepreview_postgresflexalpha_instance.ptlsdbsrv.project_id
#   instance_id = stackitprivatepreview_postgresflexalpha_instance.ptlsdbsrv.instance_id
#   username    = var.db_username
#   roles       = ["createdb", "login"]
#   # roles     = ["createdb", "login", "createrole"]
# }
#
# resource "stackitprivatepreview_sqlserverflexalpha_user" "ptlsdbuser" {
#   project_id  = stackitprivatepreview_sqlserverflexalpha_instance.ptlsdbsqlsrv.project_id
#   instance_id = stackitprivatepreview_sqlserverflexalpha_instance.ptlsdbsqlsrv.instance_id
#   username    = var.db_username
#   roles       = ["login"]
# }
