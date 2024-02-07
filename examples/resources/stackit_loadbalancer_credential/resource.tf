resource "stackit_loadbalancer_credential" "example" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  display_name = "example-credentials"
  username     = "example-user"
  password     = "example-password"
}
