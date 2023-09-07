resource "stackit_mariadb_instance" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-instance"
  version    = "10"
  plan_name  = "example-plan-name"
  parameters = {
    sgw_acl = "x.x.x.x/x,y.y.y.y/y"
  }
}
