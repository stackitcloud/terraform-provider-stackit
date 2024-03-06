resource "stackit_postgresflex_user" "example" {
  project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  description   = "Example user"
  write_enabled = false
}
