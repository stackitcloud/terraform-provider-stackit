resource "stackit_sqlserverflex_instance" "example" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example-instance"
  acl             = ["XXX.XXX.XXX.X/XX", "XX.XXX.XX.X/XX"]
  backup_schedule = "00 00 * * *"
  flavor = {
    cpu = 4
    ram = 16
  }
  replicas = 3
  storage = {
    class = "class"
    size  = 5
  }
  version = 2022
}
