resource "stackit_sqlserverflex_database" "example" {
  project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name          = "example-database"
  owner         = "example-user"
  collation     = "SQL_Latin1_General_CP1_CI_AS"
  compatibility = 160
}
