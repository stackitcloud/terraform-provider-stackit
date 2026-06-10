data "stackit_dremio_user" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region      = "eu01"
  instance_id = "example-instance-id"
  user_id = "example-user-id"
}