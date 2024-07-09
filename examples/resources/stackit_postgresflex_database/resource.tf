resource "stackit_postgresflex_database" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "mydb"
  owner       = "myusername"
}
